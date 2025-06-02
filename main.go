package main

import (
	"github.com/OliverSchlueter/goutils/sloki"
	"log/slog"
	"mail-server/smtp"
)

const hostname = "foo.com"

func main() {
	lokiService := sloki.NewService(sloki.Configuration{
		URL:          "http://localhost:3100/loki/api/v1/push",
		Service:      "mail-server",
		ConsoleLevel: slog.LevelDebug,
		LokiLevel:    slog.LevelInfo,
		EnableLoki:   false,
	})
	slog.SetDefault(slog.New(lokiService))

	smtpSever := smtp.NewServer(smtp.Configuration{
		Hostname: hostname,
		Port:     "2525",
	})
	go smtpSever.StartServer()
	slog.Info("Started SMTP server")

	c := make(chan struct{})
	<-c
}
