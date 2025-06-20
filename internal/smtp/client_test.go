package smtp

import (
	"fmt"
	"github.com/OliverSchlueter/mail-server/internal/mails"
	mdb "github.com/OliverSchlueter/mail-server/internal/mails/database/fake"
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
		Password:     users.Hash("oliver123"),
		PrimaryEmail: "oliver@localhost",
		Emails:       []string{"oliver@localhost"},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	u, err := us.GetByName("oliver")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	ms := mails.NewStore(mails.Configuration{
		DB: mdb.NewDB(),
	})

	err = ms.CreateMailbox(mails.Mailbox{
		UserID: u.ID,
		Name:   mails.DefaultMailboxName,
		UID:    mails.DefaultMailboxUID,
		Flags:  []string{},
	})
	if err != nil {
		t.Fatalf("Failed to create mailbox: %v", err)
	}

	srv := NewServer(Configuration{
		Hostname: "localhost",
		Port:     "2525",
		Users:    *us,
		Mails:    *ms,
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

	gotMails, err := ms.GetMails("oliver", mails.DefaultMailboxUID)
	if err != nil {
		t.Fatalf("Failed to get mails: %v", err)
	}

	if len(gotMails) != 1 {
		t.Errorf("Expected 1 mail in mailbox, but got %d", len(gotMails))
	}
}
