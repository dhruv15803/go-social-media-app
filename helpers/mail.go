package helpers

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/dhruv15803/social-media-app/storage"
	"gopkg.in/gomail.v2"
)

func sendGoMail(fromEmail string, subject string, user storage.User, token string, htmlTemplatePath string) error {

	verificationUrl := fmt.Sprintf("%s/verify-email?token=%s", os.Getenv("CLIENT_URL"), token)

	type MailData struct {
		Username        string
		VerificationURL string
	}

	tmpl := template.Must(template.ParseFiles(htmlTemplatePath))
	var body bytes.Buffer

	if err := tmpl.Execute(&body, MailData{Username: user.Username, VerificationURL: verificationUrl}); err != nil {
		return err
	}

	goMailUsername := os.Getenv("GOMAIL_USERNAME")
	goMailAppPassword := os.Getenv("GOMAIL_APP_PASSWORD")

	message := gomail.NewMessage()

	message.SetHeader("From", fromEmail)
	message.SetHeader("To", user.Email)
	message.SetHeader("Subject", subject)
	message.SetBody("text/html", body.String())

	dialer := gomail.NewDialer("smtp.gmail.com", 587, goMailUsername, goMailAppPassword)

	return dialer.DialAndSend(message)
}

func SendMailWithRetry(fromEmail string, subject string, user storage.User, token string, htmlTemplatePath string, maxRetries int) error {

	isMailSent := false

	for retryCount := 1; retryCount <= maxRetries; retryCount++ {

		if err := sendGoMail(fromEmail, subject, user, token, htmlTemplatePath); err != nil {
			log.Printf("failed to send email to %v , attempt - %v", user.Email, retryCount)
			continue
		}
		isMailSent = true
		break
	}

	if isMailSent {
		return nil
	} else {
		return fmt.Errorf("failed to send email to %v", user.Email)
	}
}
