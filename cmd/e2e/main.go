package main

import (
	"github.com/OliverSchlueter/goutils/sloki"
	"github.com/OliverSchlueter/mail-server/internal/imap"
	"github.com/OliverSchlueter/mail-server/internal/mails"
	fake2 "github.com/OliverSchlueter/mail-server/internal/mails/database/fake"
	"github.com/OliverSchlueter/mail-server/internal/smtp"
	"github.com/OliverSchlueter/mail-server/internal/users"
	"github.com/OliverSchlueter/mail-server/internal/users/database/fake"
	"log/slog"
)

const hostname = "localhost"

func main() {
	lokiService := sloki.NewService(sloki.Configuration{
		URL:          "http://localhost:3100/loki/api/v1/push",
		Service:      "mail-server",
		ConsoleLevel: slog.LevelDebug,
		LokiLevel:    slog.LevelInfo,
		EnableLoki:   false,
	})
	slog.SetDefault(slog.New(lokiService))

	// users
	us := users.NewStore(users.Configuration{
		DB: fake.NewDB(),
	})

	// add test users
	_ = us.Create(users.User{
		Name:         "oliver",
		Password:     "oliver123",
		PrimaryEmail: "oliver@" + hostname,
		Emails: []string{
			"oliver@" + hostname,
		},
	})

	// mails

	ms := mails.NewStore(mails.Configuration{
		DB: fake2.NewDB(),
	})

	// smtp server
	smtpSever := smtp.NewServer(smtp.Configuration{
		Hostname: hostname,
		Port:     "25",
		Users:    *us,
		Mails:    *ms,
	})
	go smtpSever.Start()
	slog.Info("Started SMTP server")

	// imap server
	imapServer := imap.NewServer(imap.Configuration{
		Port:  "143",
		Users: *us,
	})
	go imapServer.Start()
	slog.Info("Started IMAP server")

	c := make(chan struct{})
	<-c
}
