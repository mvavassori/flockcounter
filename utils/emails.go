package utils

import (
	// "context"
	"crypto/tls"
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
	"os"

	"github.com/joho/godotenv"
)

// const senderEmail = "mv@marcovassori.com"
const senderEmail = "mv@purelymail.com"

var senderPassword string

func init() {
	// load env variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	senderPassword = os.Getenv("SENDER_PASSWORD")
}

const smtpServer = "smtp.purelymail.com"
const smtpPort = "465"

func SendEmail(recipientEmail, subject, body string) error {

	log.Println("Starting SendEmail function")
	from := mail.Address{Name: "Bare Analytics", Address: senderEmail}

	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: smtpServer,
	}

	// Connect to the SMTP Server
	conn, err := tls.Dial("tcp", smtpServer+":"+smtpPort, tlsConfig)
	if err != nil {
		log.Printf("Failed to connect to SMTP server: %v", err)
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpServer)
	if err != nil {
		log.Printf("Failed to create SMTP client: %v", err)
		return err
	}
	defer client.Quit()

	// Authenticate
	auth := smtp.PlainAuth("", senderEmail, senderPassword, smtpServer)
	log.Printf("Attempting authentication with email: %s", senderEmail)
	// WARNING: Be careful with logging passwords in production environments
	log.Printf("Password used: %s", senderPassword)
	if err = client.Auth(auth); err != nil {
		log.Printf("Authentication failed. Error: %v", err)
		// Log more details about the error if possible
		return err
	}
	log.Println("Authentication successful")

	// Set the sender and recipient
	if err = client.Mail(from.Address); err != nil {
		log.Printf("Failed to set sender: %v", err)
		return err
	}
	if err = client.Rcpt(recipientEmail); err != nil {
		log.Printf("Failed to set recipient: %v", err)
		return err
	}

	// Send the email body
	writer, err := client.Data()
	if err != nil {
		log.Printf("Failed to open data writer: %v", err)
		return err
	}
	defer writer.Close()

	header := make(map[string]string)
	header["From"] = from.String()
	header["To"] = recipientEmail
	header["Subject"] = subject

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	_, err = writer.Write([]byte(message))
	if err != nil {
		log.Printf("Failed to write email body: %v", err)
		return err
	}

	log.Println("Email sent successfully!")
	return nil
}
