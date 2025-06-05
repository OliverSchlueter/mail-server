package smtp

import (
	"fmt"
	"github.com/OliverSchlueter/mail-server/internal/users"
	"github.com/OliverSchlueter/mail-server/internal/users/database/fake"
	"log/slog"
	"testing"
)

func TestSendMail(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	us := users.NewStore(users.Configuration{
		DB: fake.NewDB(),
	})
	err := us.Create(users.User{
		Name:         "oliver",
		Password:     "oliver123",
		PrimaryEmail: "oliver@localhost",
		Emails:       []string{"oliver@localhost"},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	srv := NewServer(Configuration{
		Hostname: "localhost",
		Port:     "25",
		Users:    *us,
	})
	go srv.Start()
	fmt.Printf("SMTP server started on %s:%s\n", srv.hostname, srv.port)

	mail := Mail{
		Outgoing:    true,
		From:        "peter@fancyinnovations.com",
		To:          []string{"oliver@localhost"},
		DataBuffer:  []string{"Subject: Test Mail", "", "This is a test mail."},
		ReadingData: false,
	}

	n, err := SendMail(mail)
	if err != nil {
		t.Errorf("Failed to send mail: %v", err)
	}

	if n != 1 {
		t.Errorf("Expected to send 1 email, but sent %d", n)
	}
}
