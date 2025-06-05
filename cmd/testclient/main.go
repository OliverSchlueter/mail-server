package main

import (
	"github.com/OliverSchlueter/mail-server/internal/smtp"
	"github.com/wneessen/go-mail"
	"log"
	"log/slog"
	"time"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	//incomingMail()
	//outgoingMail()
	ourClient()
}

func ourClient() {
	n, err := smtp.SendMail(smtp.Mail{
		Outgoing: true,
		From:     "mail@schlueter-oliver.de",
		To: []string{
			"oliver@fancyinnovations.com",
		},
		DataBuffer: []string{
			"Date: " + time.Now().Format(time.RFC1123Z),
			"MIME-Version: 1.0",
			"Message-ID: <" + time.Now().Format("20060102150405") + "@schlueter-oliver.de>",
			"Subject: Test Mail",
			"From: <mail@schlueter-oliver.de>",
			"To: <oliver@fancyinnovations.com>",
			"Content-Transfer-Encoding: quoted-printable",
			"Content-Type: text/plain; charset=UTF-8",

			"",
			"This is a test mail.",
		},
		ReadingData: false,
	})
	if err != nil {
		log.Fatalf("failed to send mail: %s", err)
	}

	log.Printf("Sent %d emails", n)
}

func outgoingMail() {
	// First we create a mail message
	m := mail.NewMsg()
	if err := m.From("oliver@localhost"); err != nil {
		log.Fatalf("failed to set From address: %s", err)
	}
	if err := m.To("peter@otherdomain.com"); err != nil {
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
		mail.WithPassword("oliver123"),
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

func incomingMail() {
	// First we create a mail message
	m := mail.NewMsg()
	if err := m.From("peter@otherdomain.com"); err != nil {
		log.Fatalf("failed to set From address: %s", err)
	}
	if err := m.To("oliver@localhost"); err != nil {
		log.Fatalf("failed to set To address: %s", err)
	}
	m.Subject("Why are you not using go-mail yet?")
	m.SetBodyString(mail.TypeTextPlain, "You won't need a sales pitch. It's FOSS.")

	// Secondly the mail client
	c, err := mail.NewClient(
		"localhost",
		mail.WithPort(2525),
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
