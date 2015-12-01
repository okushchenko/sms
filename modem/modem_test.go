package modem

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

type FakePort struct {
	buffer []byte
	reader *bytes.Reader
}

func (p *FakePort) Read(b []byte) (n int, err error) {
	if p.reader == nil {
		p.reader = bytes.NewReader(p.buffer)
	}
	n, err = p.reader.Read(b)
	return
}

func (p *FakePort) Write(b []byte) (n int, err error) {
	InitCommands := map[string]string{
		"ATZ\r":                          "\r\nOK\r\n",
		"AT\r":                           "\r\nOK\r\n",
		"ATE0\r":                         "ATE0\r\nOK\r\n",
		"AT+CFUN=1\r":                    "\r\nOK\r\n",
		"AT+CMEE=1\r":                    "\r\nOK\r\n",
		"AT+COPS=3,0\r":                  "\r\nOK\r\n",
		"AT+CMGF=0\r":                    "\r\nOK\r\n",
		"AT+CMGF=1\r":                    "\r\nOK\r\n",
		"AT^USSDMODE=1\r":                "\r\nOK\r\n",
		"AT+CUSD=1,\"AA582C3602\",15\r":  "\r\nFFFFFFFFFFFFFFFFFFFFFFFF\r\nOK\r\n+CUSD: 0,\"C2303BEC9E8362B09B0B0643CBDD2C90F8EDAECF4130170C8696BB5D0A954AA58096E5657B5ABE0E83F461767E8E5ED741F0F79C5D3F835431596CA400\",15\r\n",
		"AT+CSMP=49,167,0,0\r":           "\r\nOK\r\n",
		"AT+CPMS=\"ME\",\"ME\",\"ME\"\r": "\r\n+CPMS: 23,50,23,50,23,50\r\n\r\nOK\r\n",
		"AT+CNMI=2,1,0,2\r":              "\r\nOK\r\n",
		"AT+CSQ\r":                       "\r\n+CSQ: 23,99\r\n\r\nOK\r\n",
		"AT+CSCS?\r":                     "\r\n+CSCS: \"IRA\"\r\n\r\nOK\r\n",
		"AT+CMGD=?\r":                    "\r\n+CMGD: (0,3,17),(0-4)\r\n\r\nOK\r\n",
		"AT+CMGD=0\r":                    "\r\nOK\r\n",
		"AT+CMGR=0\r":                    "\r\n+CMGR: \"REC UNREAD\",\"1081051021015841\",,\"15/11/02,17:34:06+08\"\r\n041404170412041E041D04060422042C0020041704100020041A041E04200414041E041D002004140415042804150412041E00210020040404320440043E043F0430002C00200410043C043504400438043A0430002C0020041A0438044204300439002C00200420043E04410456044F00200442043000200456043D044804560020043A0440\r\n\r\nOK\r\n",
		"AT+CMGR=3\r":                    "\r\n+CMGR: \"REC READ\",\"53525151\",,\"15/10/29,17:49:08+08\"\r\n42616C616E732034362E303068726E2C20626F6E757320302E303068726E2E0A2A2A2A0A5A616C7973686F6B207363686F64656E6E6F676F2070616B65747520706F736C75673A203435534D533B2042657A6C696D69746E69206876796C796E79206E61206C6966653A293B2035302E304D4220496E7465726E6574753B20447A76696E6B7920706F203235206B6F702F6876206E6120696E\r\n\r\nOK\r\n",
		"AT+CMGR=17\r":                   "\r\n+CMGR: \"REC READ\",\"+380631234567\",,\"15/11/01,03:20:05+08\"\r\ntest\r\n\r\nOK\r\n",
		"test" + string(26):              "\r\nOK\r\n",
	}
	if InitCommands[string(b)] != "" {
		p.buffer = ([]byte(InitCommands[string(b)]))
	}
	return 0, nil
}

func (p *FakePort) Flush() error {
	p.buffer = make([]byte, 0)
	p.reader = nil
	return nil
}

func (p *FakePort) Close() (err error) {
	return nil
}

type FakeModem struct {
	ComPort string
	Port    Port
}

func FakeNew() (modem *FakeModem) {
	modem = &FakeModem{}
	return modem
}

func (m *FakeModem) Connect() (err error) {
	m.Port = &FakePort{}
	return nil
}

func TestConnect(t *testing.T) {
	gsm := FakeNew()
	err = gsm.Connect()
	if err != nil {
		t.Fatal(err)
	}
}

func TestReset(t *testing.T) {
	gsm := FakeNew()
	err = gsm.Connect()
	if err != nil {
		t.Fatal(err)
	}
	err = Reset(gsm.Port)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCheckConnection(t *testing.T) {
	gsm := FakeNew()
	gsm.Connect()
	err = CheckConnection(gsm.Port)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSignal(t *testing.T) {
	gsm := FakeNew()
	gsm.Connect()
	signal, err := GetSignal(gsm.Port)
	if err != nil {
		t.Fatal(err)
	}
	if signal != 23.99 {
		t.Fatalf("Expected 23.99, got %#v", signal)
	}
}

func TestGetCharset(t *testing.T) {
	gsm := FakeNew()
	gsm.Connect()
	charset, err := GetCharset(gsm.Port)
	if err != nil {
		t.Fatal(err)
	}
	if charset != "\"IRA\"" {
		t.Fatalf("Expected \"IRA\", got %#v", charset)
	}
}

func TestGetMessageIndexes(t *testing.T) {
	expectedIndexes := []int{0, 3, 17}
	gsm := FakeNew()
	gsm.Connect()
	indexes, err := GetMessageIndexes(gsm.Port)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indexes, expectedIndexes) {
		t.Fatalf("Expected %#v, got %#v", expectedIndexes, indexes)
	}
}

func TestGetMessage(t *testing.T) {
	expectedTime, _ := time.Parse("06/01/02,15:04:05-07", "15/10/29,17:49:08+08")
	expectedMessage := &message{
		Labels: "REC READ",
		Date:   expectedTime,
		Sender: "53525151",
		Body:   "Balans 46.00hrn, bonus 0.00hrn.\n***\nZalyshok schodennogo paketu poslug: 45SMS; Bezlimitni hvylyny na life:); 50.0MB Internetu; Dzvinky po 25 kop/hv na in",
		Index:  3,
	}
	gsm := FakeNew()
	gsm.Connect()
	message, err := GetMessage(gsm.Port, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(message, expectedMessage) {
		t.Fatalf("Expected %#v, got %#v", expectedMessage, message)
	}
}

func TestGetMessages(t *testing.T) {
	expectedTimes := make([]time.Time, 3)
	expectedTimes[0], _ = time.Parse("06/01/02,15:04:05-07", "15/11/02,17:34:06+08")
	expectedTimes[1], _ = time.Parse("06/01/02,15:04:05-07", "15/10/29,17:49:08+08")
	expectedTimes[2], _ = time.Parse("06/01/02,15:04:05-07", "15/11/01,03:20:05+08")
	expectedMessages := []*message{
		{
			Labels: "REC UNREAD",
			Date:   expectedTimes[0],
			Sender: "1081051021015841",
			Body:   "ДЗВОНІТЬ ЗА КОРДОН ДЕШЕВО! Європа, Америка, Китай, Росія та інші кр",
			Index:  0,
		},
		{
			Labels: "REC READ",
			Date:   expectedTimes[1],
			Sender: "53525151",
			Body:   "Balans 46.00hrn, bonus 0.00hrn.\n***\nZalyshok schodennogo paketu poslug: 45SMS; Bezlimitni hvylyny na life:); 50.0MB Internetu; Dzvinky po 25 kop/hv na in",
			Index:  3,
		},
		{
			Labels: "REC READ",
			Date:   expectedTimes[2],
			Sender: "+380631234567",
			Body:   "test",
			Index:  17,
		},
	}
	gsm := FakeNew()
	gsm.Connect()
	messages, err := GetMessages(gsm.Port)
	if err != nil {
		t.Fatal(err)
	}
	for i, _ := range messages {
		if !reflect.DeepEqual(messages[i], expectedMessages[i]) {
			t.Fatalf("Expected %#v\nGot %#v", expectedMessages[i], messages[i])
		}
	}
}

func TestSendMessage(t *testing.T) {
	gsm := FakeNew()
	gsm.Connect()
	err = SendMessage(gsm.Port, "+380631234567", "test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteMessage(t *testing.T) {
	gsm := FakeNew()
	gsm.Connect()
	err = DeleteMessage(gsm.Port, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNew(t *testing.T) {
	modem := New("/dev/ttyUSB0", 115200)
	expectedModem := &GSMModem{ComPort: "/dev/ttyUSB0", BaudRate: 115200}
	if !reflect.DeepEqual(modem, expectedModem) {
		t.Fatal(err)
	}
}

func TestGetBalance(t *testing.T) {
	balanceExpected := 107.0
	gsm := FakeNew()
	gsm.Connect()
	balance, err := GetBalance(gsm.Port, `*111#`)
	if err != nil {
		t.Fatal(err)
	}
	if balance != balanceExpected {
		t.Fatalf("Expected %#v\nGot %#v", balanceExpected, balance)
	}
}

// func TestSendSMS(t *testing.T) {
// 	var indexes []int
// 	var newIndexes []int
// 	gsm := New("/dev/ttyUSB0", 115200, "0")
// 	gsm.Connect()
// 	gsm.Reset()
// 	indexes, err = gsm.GetMessageIndexes()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	err = gsm.SendSMS("+380637615869", "test")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for i := 0; i < 20; i++ {
// 		newIndexes, err = gsm.GetMessageIndexes()
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		if len(newIndexes) != len(indexes) {
// 			log.Println("Received message after", float64(i)/4.0, "seconds")
// 			break
// 		} else if i < 19 {
// 			time.Sleep(time.Millisecond * 250)
// 		} else {
// 			t.Fatal("Timed out while waiting for message to arrive")
// 		}
// 	}
// 	msg, err := gsm.GetMessage(newIndexes[len(newIndexes)-1])
// 	if msg.Body != "test" {
// 		t.Fatal("Got wrong message:", msg.Body)
// 	}
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
