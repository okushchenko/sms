package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/alexgear/sms/common"
	db "github.com/alexgear/sms/database"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

var err error

//response structure to /sms
type SMSResponse struct {
	Status int    `json:"status"`
	Text   string `json:"text"`
	UUID   string `json:"uuid"`
}

func sendSMSHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("sendSMSHandler: %#v", r)
	w.Header().Set("Content-type", "application/json")

	r.ParseForm()
	mobile := r.FormValue("to")
	message := r.FormValue("text")
	uuid := uuid.NewV1()
	sms := &common.SMS{UUID: uuid.String(), Mobile: mobile, Body: message, Status: "pending"}
	err = db.InsertMessage(sms)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	smsresp := SMSResponse{Status: 200, Text: sms.Body, UUID: sms.UUID}
	var toWrite []byte
	toWrite, err := json.Marshal(smsresp)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(toWrite)
}

func InitServer(host string, port int) error {
	r := mux.NewRouter()
	r.HandleFunc("/api/sms", sendSMSHandler)

	http.Handle("/", r)

	bind := fmt.Sprintf("%s:%d", host, port)
	log.Println("listening on: ", bind)
	return http.ListenAndServe(bind, nil)

}
