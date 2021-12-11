package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"notif/implementation/email"
	"strings"

	"github.com/matcornic/hermes/v2"
)

func main() {
	// prepare toList
	toList := make([]email.NameAddr, 1)
	toList[0].EmailAddr = "michaelprimr.1119.9.5@gmail.com"
	toList[0].UserName = "Michael"

	e := email.Entity{
		FromName: "Sourik Ghosh",
		ToList:   toList,
		Subject:  "thanks for choosing notif",
	}

	// prepare html body
	e.Body = prepareCustomBeatifulHtmlBody(e)
	e.Body = buildReqBody(e)

	body, err := json.Marshal(&e)
	if err != nil {
		log.Fatalf("marshalling failed:%s", err.Error())
	}

	reqBody := bytes.NewBuffer(body)
	url := "http://localhost:6969/notif-svc/v1/create"
	contentType := "application/json"

	resp, err := http.Post(url, contentType, reqBody)
	if err != nil {
		log.Fatalf("failed to make request:%s", err.Error())
	}

	if resp.StatusCode == http.StatusOK {
		log.Println("successfully sent")
	}
}

func buildReqBody(e email.Entity) string {
	tolist := make([]string, len(e.ToList))
	for i := range e.ToList {
		tolist = append(tolist, e.ToList[i].EmailAddr)
	}

	msg := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg += fmt.Sprintf("From: %s\r\n", e.FromName)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(tolist, ";"))
	msg += fmt.Sprintf("Subject: %s\r\n", e.Subject)
	msg += fmt.Sprintf("\r\n%s\r\n", e.Body)

	return msg
}

func prepareCustomBeatifulHtmlBody(e email.Entity) string {
	h := hermes.Hermes{
		Theme: new(hermes.Default),
		Product: hermes.Product{
			Copyright: "Copyright © 2021 Notif. All rights reserved.",
			Name:      "Notif",
			Logo:      "https://storage.googleapis.com/gopherizeme.appspot.com/gophers/8f42d0b66299ee3f295d8299da93b8c60fba2bd8.png",
		},
	}

	// will panic if len(toList) == 0 !!
	usr := e.ToList[0].UserName

	email := hermes.Email{
		Body: hermes.Body{
			Greeting: "Hey",
			Name:     usr,
			Intros: []string{
				"Welcome to Notif! We're very excited to have you on board.",
			},

			Actions: []hermes.Action{
				{
					Instructions: "To get started with Notif, please click here:",
					Button: hermes.Button{
						Color: "#1FB884",
						Text:  "Confirm your account",
						Link:  "https://github.com/sourikghosh/notif",
					},
				},
			},
			Outros: []string{
				"Need help, or have questions? Just reply to this email, we'd love to help.",
			},
			Signature: "Thanks",
		},
	}

	bodyStr, err := h.GenerateHTML(email)
	if err != nil {
		return ""
	}

	return bodyStr
}
