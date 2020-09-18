package notification

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"text/template"

	"github.com/joho/godotenv"
)

// SendEmail ... Function to send email
func SendEmail(to []string, subject string, templateHTML string, bodyVar interface{}, funcMap map[string]interface{}) {

	_ = godotenv.Load()

	var mailServerUsername string = os.Getenv("SMTPUser")
	var mailServerPassword string = os.Getenv("SMTPPassword")
	const sender string = "noreply@sidomuncul.co.id"

	const headers string = "MIME-version: 1.0;\nContent-Type: text/html;"

	t, err := template.ParseFiles("./notification/template/" + templateHTML)
	if err != nil {
		log.Println(err.Error())
	}

	t.Funcs(funcMap)

	var body bytes.Buffer

	gmailAuth := smtp.PlainAuth("", mailServerUsername, mailServerPassword, "smtp-relay.gmail.com")

	body.Write([]byte(fmt.Sprintf("From: %s\nSubject: %s\n%s\n\n", sender, subject, headers)))

	err = t.Execute(&body, bodyVar)
	if err != nil {
		log.Println(err.Error())
	}
	err = smtp.SendMail("smtp-relay.gmail.com:587", gmailAuth, sender, to, body.Bytes())
	if err != nil {
		log.Println(err.Error())
	}
}
