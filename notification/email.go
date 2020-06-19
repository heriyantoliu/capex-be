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
func SendEmail(to []string, subject string, templateHTML string, bodyVar interface{}) {

	_ = godotenv.Load()

	var mailServerUsername string = os.Getenv("SMTPUser")
	var mailServerPassword string = os.Getenv("SMTPPassword")
	const sender string = "noreply@sidomuncul.co.id"

	const headers string = "MIME-version: 1.0;\nContent-Type: text/html;"

	gmailAuth := smtp.PlainAuth("", mailServerUsername, mailServerPassword, "smtp-relay.gmail.com")
	t, err := template.ParseFiles("./notification/template/" + templateHTML)
	if err != nil {
		log.Println(err.Error())
	}

	var body bytes.Buffer

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
