package worker

import (
	"log"
	"time"

	"github.com/alexgear/sms/common"
	"github.com/alexgear/sms/database"
	"github.com/alexgear/sms/modem"
)

var err error

func InitWorker() {
	messages := make(chan common.SMS)
	go producer(messages)
	go consumer(messages)
}

func consumer(messages chan common.SMS) {
	for {
		message := <-messages
		log.Println("consumer: processing", message.UUID)
		err = modem.SendMessage(message.Mobile, message.Body)
		if err != nil {
			message.Status = "error"
			log.Println("consumer: failed to process", message.UUID, err)
		} else {
			message.Status = "sent"
		}
		message.Retries++
		// TODO: make this update a goroutine?
		database.UpdateMessageStatus(message)
	}
}

func producer(messages chan common.SMS) {
	for {
		pendingMsgs, err := database.GetPendingMessages()
		if err != nil {
			log.Println("producer: failed to get messages. %s", err.Error())
		}
		log.Printf("producer: %d pending messages found", len(pendingMsgs))
		for _, msg := range pendingMsgs {
			log.Printf("producer: Processing %#v", msg)
			messages <- msg
		}
		time.Sleep(10000 * time.Millisecond)
	}
}
