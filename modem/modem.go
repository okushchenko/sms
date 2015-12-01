package modem

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
	pdu "github.com/xlab/at/pdu"
)

var err error

type Modem interface {
	Connect() (err error)
}

type GSMModem struct {
	ComPort  string
	BaudRate int
	Port     Port
}

type message struct {
	Labels string
	Sender string
	Date   time.Time
	Body   string
	Index  int
}

type Port interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Flush() error
	Close() (err error)
}

func New(ComPort string, BaudRate int) (modem *GSMModem) {
	modem = &GSMModem{ComPort: ComPort, BaudRate: BaudRate}
	return modem
}

func (m *GSMModem) Connect() (err error) {
	config := &serial.Config{Name: m.ComPort, Baud: m.BaudRate, ReadTimeout: time.Second}
	m.Port, err = serial.OpenPort(config)
	if err != nil {
		return fmt.Errorf("GSMModem.Connect: Failed to open port. %s", err.Error())
	}
	return nil
}

func SendCommand(p Port, command string, wait bool) (string, error) {
	log.Println("SendCommand...", command)
	p.Flush()
	_, err = p.Write([]byte(command))
	if err != nil {
		return "", fmt.Errorf("SendCommand: Failed to write to port.\n%s", err.Error())
	}
	var reps int = 1
	if wait {
		reps = 5
	}
	output, err := WaitForOutput(p, reps, "OK\r\n")
	if err != nil {
		return "", fmt.Errorf("SendCommand: Failed to wait for output.\n%s", err.Error())
	}
	return output, nil
}

func WaitForOutput(p Port, reps int, suffix string) (string, error) {
	var status string
	var buffer bytes.Buffer
	buf := make([]byte, 32)
	for i := 0; i < reps; {
		// ignoring error as EOF raises error on Linux
		n, _ := p.Read(buf)
		if n > 0 {
			buffer.Write(buf[:n])
			status = buffer.String()
			log.Printf("WaitForOutput: received %d bytes: %#v\n", n, string(buf[:n]))
			if strings.HasSuffix(status, suffix) {
				return status, nil
			} else if regexp.MustCompile(`[A-Z ]*ERROR[0-9A-Za-z ]*`).MatchString(status) {
				errorCodes := regexp.MustCompile(`([A-Z ]*)ERROR([0-9A-Za-z :]*)`).FindAllStringSubmatch(status, -1)
				if errorCodes[0][1] == "" && errorCodes[0][2] == "" {
					return status, fmt.Errorf("WaitForOutput: Found unknown ERROR")
				} else {
					return status, fmt.Errorf("WaitForOutput: Found %vERROR%v", errorCodes[0][1], errorCodes[0][2])
				}
			}
		} else {
			log.Printf("WaitForOutput: No output on %d iteration", i)
			// time.Sleep(time.Millisecond * 500)
			i++
		}
	}
	return status, errors.New("WaitForOutput: Timed out.")
}

func GetSignal(p Port) (float64, error) {
	log.Println("GetSignal...")
	status, err := SendCommand(p, "AT+CSQ\r", true)
	if err != nil {
		return 0.0, err
	}
	return strconv.ParseFloat(
		strings.Replace(
			regexp.MustCompile(`\d+,\d+`).FindString(status), ",", ".", 1), 64)
}

func GetCharset(p Port) (string, error) {
	log.Println("GetCharset...")
	status, err := SendCommand(p, "AT+CSCS?\r", true)
	if err != nil {
		return "", err
	}
	return regexp.MustCompile(`\"[A-Za-z0-9]+\"`).FindString(status), nil
}

func CheckConnection(p Port) error {
	log.Println("CheckConnection...")
	_, err = SendCommand(p, "AT\r", true)
	if err != nil {
		return err
	}
	return nil
}

func Reset(p Port) error {
	log.Println("Reset...")
	InitCommands := []string{
		"ATZ\r",
		"ATE0\r",
		"AT+CFUN=1\r",
		"AT+CMEE=1\r",
		"AT+COPS=3,0\r",
		"AT+CMGF=1\r",
		"AT+CSMP=49,167,0,0\r",
		"AT+CPMS=\"ME\",\"ME\",\"ME\"\r",
		"AT+CNMI=2,1,0,2\r",
	}
	for _, c := range InitCommands {
		for i := 0; i < 10; i++ {
			log.Printf("%v, %#v", i, c)
			_, err = SendCommand(p, c, false)
			if err != nil && i < 9 {
				log.Println(err)
				time.Sleep(time.Millisecond * 500)
			} else if err != nil && i == 9 {
				return err
			} else {
				break
			}
		}
	}
	return nil
}

func GetBalance(p Port, ussdRequest string) (float64, error) {
	log.Println("GetBalance...")
	//re-set encoding here?
	//m.SendCommand("AT+CSCS=\"GSM\"\r", false)
	//TODO: Is it necessery to run AT+CMGF=0 ???
	SendCommand(p, "AT+CMGF=0\r", false)
	SendCommand(p, "AT^USSDMODE=1\r", false)
	request := strings.ToUpper(fmt.Sprintf("%x", pdu.Encode7Bit(ussdRequest)))
	_, err = SendCommand(p, fmt.Sprintf("AT+CUSD=1,\"%s\",15\r", request), true)
	if err != nil {
		return 0.0, err
	}
	status, err := WaitForOutput(p, 10, "15\r\n")
	regex := regexp.MustCompile(`\+CUSD: \d{1},\"([a-zA-Z0-9]*)\",\d*`)
	if regex.MatchString(status) {
		balanceRaw := regex.FindStringSubmatch(status)[1]
		bytesWritten, _ := hex.DecodeString(balanceRaw)
		log.Println("Before decode", bytesWritten)
		balanceRaw, _ = pdu.Decode7Bit(bytesWritten)
		log.Println("After decode", balanceRaw)
		balanceParsed := regexp.MustCompile(`\d+\.\d+`).FindString(balanceRaw)
		if balanceParsed == "" {
			return 0.0, fmt.Errorf("GetBalance: Failed to find balance string in \"%s\"", balanceRaw)
		}
		balance, err := strconv.ParseFloat(balanceParsed, 64)
		if err != nil {
			return 0.0, fmt.Errorf("GetBalance: Failed to convert to float64 \"%s\"", balanceRaw)
		}
		return balance, nil
	}
	if err != nil {
		return 0.0, err
	}
	return 0.0, errors.New("GetBalace: Failed to get balance.")
}

func SendMessage(p Port, mobile string, message string) error {
	log.Println("SendMessage...", mobile, message)
	// Put Modem in SMS Text Mode
	SendCommand(p, "AT+CMGF=1\r", false)
	// Send message
	SendCommand(p, "AT+CMGS=\""+mobile+"\"\r", false)
	// EOM CTRL-Z = 26
	_, err := SendCommand(p, message+string(26), true)
	if err != nil {
		return fmt.Errorf("SendMessage: Failed to send command.\n%s", err.Error())
	}
	return nil
}

func DeleteMessage(p Port, messageIndex int) error {
	log.Println("DeleteMessage...")
	// Put Modem in SMS Text Mode
	SendCommand(p, "AT+CMGF=1\r", false)
	_, err := SendCommand(p, fmt.Sprintf("AT+CMGD=%d\r", messageIndex), true)
	if err != nil {
		return fmt.Errorf("DeleteMessage: Failed to send command.\n%s", err.Error())
	}
	return nil
}

func GetMessage(p Port, messageIndex int) (*message, error) {
	log.Println("GetMessage...")
	status, err := SendCommand(p, fmt.Sprintf("AT+CMGR=%d\r", messageIndex), true)
	if err != nil {
		return nil, fmt.Errorf("GetMessage: Failed to send command.\n%s", err.Error())
	}
	log.Printf("GetMessage: %#v\n", status)
	regex := regexp.MustCompile(`(?Us)CMGR: "([A-Z ]*)","([+\d]*)",,"([0-9/,:\+]*)"\r\n(.*)\r\n\r\nOK`)
	if regex.MatchString(status) {
		msg := regex.FindStringSubmatch(status)
		messageLabels := msg[1]
		messageSender := msg[2]
		messageDate, _ := time.Parse("06/01/02,15:04:05-07", msg[3])
		var messageBody string
		if regexp.MustCompile(`^[0-9A-F]+$`).MatchString(msg[4]) {
			bytesWritten, _ := hex.DecodeString(msg[4])
			log.Println("GetMessage: bytesWritten =", bytesWritten)
			regex := regexp.MustCompile(`[a-z ]{3}`)
			if regex.MatchString(string(bytesWritten)) {
				log.Printf("GetMessage: Decoding message #%d using plain text", messageIndex)
				messageBody = string(bytesWritten)
			} else {
				log.Printf("GetMessage: Decoding message #%d using Ucs2", messageIndex)
				messageBody, err = pdu.DecodeUcs2(bytesWritten)
				if err != nil {
					log.Printf("GetMessage: Failed to decode message #%d using Ucs2", messageIndex)
				}
			}
		} else {
			messageBody = msg[4]
		}
		log.Printf("GetMessage: %v %#v %#v %v %#v\n", messageIndex, messageLabels, messageSender, messageDate.Format(time.RFC3339), messageBody)
		return &message{
			Labels: messageLabels,
			Date:   messageDate,
			Sender: messageSender,
			Body:   messageBody,
			Index:  messageIndex,
		}, nil
	} else {
		return nil, fmt.Errorf("GetMessage: Failed to parse response: %v", status)
	}
}

func GetMessageIndexes(p Port) ([]int, error) {
	var messageIndexes []int
	log.Println("GetMessageIndexes...")
	// Put Modem in SMS Text Mode
	SendCommand(p, "AT+CMGF=1\r", false)
	// Get message indexes
	status, err := SendCommand(p, "AT+CMGD=?\r", true)
	if err != nil {
		return messageIndexes, err
	}
	regex := regexp.MustCompile(`\+CMGD: \(([0-9,]*)\)`)
	if regex.MatchString(status) {
		var messageIndexesRaw []string
		statusParsed := regex.FindStringSubmatch(status)[1]
		if statusParsed != "" {
			messageIndexesRaw = strings.Split(statusParsed, ",")
		}
		for _, messageIndex := range messageIndexesRaw {
			index, err := strconv.Atoi(messageIndex)
			if err != nil {
				log.Printf("GetMessages: Failed to convert messageIndex=\"%v\" to int", messageIndex)
			} else {
				messageIndexes = append(messageIndexes, index)
			}
		}
		return messageIndexes, nil
	} else {
		return nil, errors.New("GetMessageIndexes: Failed to get message indexes")
	}
}

func GetMessages(p Port) ([]*message, error) {
	log.Println("GetMessages...")
	var messages []*message
	messageIndexes, err := GetMessageIndexes(p)
	if err != nil {
		return messages, err
	}
	log.Println("GetMessages:", messageIndexes)
	for _, messageIndex := range messageIndexes {
		msg, err := GetMessage(p, messageIndex)
		if err != nil {
			return messages, err
		} else {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}
