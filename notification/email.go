package notification

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
	"text/template"
)

// func SendEmail(to []string, subject string, templateHTML string, bodyVar map[string]string) {
// 	from := "fotocopy@sidomuncul.co.id"
// 	pass := "SidoMuncul2018"
// 	// to1 := "heriyanto.liu@sidomuncul.co.id"
// 	sender := "no-reply@sidomuncul.co.id"

// 	toHeader := strings.Join(to, ",")

// 	msg := "From: " + sender + "\n" +
// 		"To: " + toHeader + "\n" +
// 		"Subject: " + subject + "\n\n" +
// 		message

// 	err := smtp.SendMail("smtp-relay.gmail.com:587",
// 		smtp.PlainAuth("", from, pass, "smtp-relay.gmail.com"),
// 		from, to, []byte(msg))

// 	if err != nil {
// 		log.Printf("smtp error: %s", err)
// 		return
// 	}
// 	for _, toEmail := range to {
// 		log.Println("email sent to:" + toEmail)
// 	}

// }

// SendEmail ... Function to send email
func SendEmail(to []string, subject string, templateHTML string, bodyVar map[string]string) {

	const mailServerUsername string = "fotocopy@sidomuncul.co.id"
	const mailServerPassword string = "SidoMuncul2018"
	const sender string = "noreply@sidomuncul.co.id"

	const headers string = "MIME-version: 1.0;\nContent-Type: text/html;"

	gmailAuth := smtp.PlainAuth("", mailServerUsername, mailServerPassword, "smtp-relay.gmail.com")
	t, err := template.ParseFiles("./notification/template/" + templateHTML)
	if err != nil {
		log.Println(err.Error())
	}

	var body bytes.Buffer

	body.Write([]byte(fmt.Sprintf("From: %s\nSubject: %s\n%s\n\n", sender, subject, headers)))

	err = t.Execute(&body, struct {
		Name    string
		CapexID string
		Message string
	}{
		Name:    bodyVar["Name"],
		CapexID: bodyVar["CapexID"],
		Message: bodyVar["Message"],
	})
	if err != nil {
		log.Println(err.Error())
	}
	err = smtp.SendMail("smtp-relay.gmail.com:587", gmailAuth, sender, to, body.Bytes())
	if err != nil {
		log.Println(err.Error())
	}
}
