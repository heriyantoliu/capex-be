package notification

import (
	"bytes"
	"log"
	"os"
	"text/template"

	"gopkg.in/gomail.v2"
)

// SendEmail ... Function to send email
func SendEmail(to []string, cc []string, subject string, templateHTML string, bodyVar interface{}, funcMap map[string]interface{}) {

	var smtpUser string = os.Getenv("SMTPUser")
	var smtpPass string = os.Getenv("SMTPPassword")

	t := template.New(templateHTML)

	t, err := template.ParseFiles("./notification/template/" + templateHTML)
	if err != nil {
		log.Println(err.Error())
	}

	var tpl bytes.Buffer

	err = t.Execute(&tpl, bodyVar)
	if err != nil {
		log.Println(err.Error())
	}

	result := tpl.String()
	m := gomail.NewMessage()
	m.SetHeader("From", smtpUser)

	for _, address := range to {
		if address != "" {
			m.SetHeader("To", address)
		}
	}

	for _, address := range cc {
		if address != "" {
			m.SetAddressHeader("Cc", address, "")
		}
	}

	m.SetHeader("Subject", subject)

	m.SetBody("text/html", result)

	d := gomail.NewDialer("smtp.gmail.com", 587, smtpUser, smtpPass)

	err = d.DialAndSend(m)
	if err != nil {
		log.Println(err)
	}

}
