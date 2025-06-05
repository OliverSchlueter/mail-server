package main

import (
	"github.com/wneessen/go-mail"
	"log"
)

func main() {
	// First we create a mail message
	m := mail.NewMsg()
	if err := m.From("oliver@fancyinnovations.com"); err != nil {
		log.Fatalf("failed to set From address: %s", err)
	}
	if err := m.To("peter@fancyinnovations.com"); err != nil {
		log.Fatalf("failed to set To address: %s", err)
	}
	m.Subject("Why are you not using go-mail yet?")
	m.SetBodyString(mail.TypeTextPlain, "You won't need a sales pitch. It's FOSS.")

	// Secondly the mail client
	c, err := mail.NewClient(
		"localhost",
		mail.WithPort(2525),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername("oliver"),
		mail.WithPassword("oliver"),
		mail.WithTLSPolicy(mail.NoTLS),
	)
	if err != nil {
		log.Fatalf("failed to create mail client: %s", err)
	}

	// Finally let's send out the mail
	if err := c.DialAndSend(m); err != nil {
		log.Fatalf("failed to send mail: %s", err)
	}
}
