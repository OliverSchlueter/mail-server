package fake

import (
	"github.com/OliverSchlueter/mail-server/internal/mails"
	"sync"
)

type DB struct {
	Mailboxes []mails.Mailbox
	Mails     []mails.Mail
	mu        sync.Mutex
}

func NewDB() *DB {
	return &DB{
		Mailboxes: []mails.Mailbox{},
		Mails:     []mails.Mail{},
		mu:        sync.Mutex{},
	}
}

func (db *DB) GetMailboxes(userID string) ([]mails.Mailbox, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var userMailboxes []mails.Mailbox
	for _, mailbox := range db.Mailboxes {
		if mailbox.UserID == userID {
			userMailboxes = append(userMailboxes, mailbox)
		}
	}
	return userMailboxes, nil
}

func (db *DB) GetMailboxByUID(userID string, uid uint32) (*mails.Mailbox, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, mailbox := range db.Mailboxes {
		if mailbox.UserID == userID && mailbox.UID == uid {
			return &mailbox, nil
		}
	}
	return nil, mails.ErrMailboxNotFound
}

func (db *DB) GetMailboxByName(userID string, name string) (*mails.Mailbox, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, mailbox := range db.Mailboxes {
		if mailbox.UserID == userID && mailbox.Name == name {
			return &mailbox, nil
		}
	}
	return nil, mails.ErrMailboxNotFound
}

func (db *DB) InsertMailbox(mailbox mails.Mailbox) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if mailbox already exists
	for _, existing := range db.Mailboxes {
		if existing.UserID == mailbox.UserID && existing.Name == mailbox.Name {
			return mails.ErrMailboxAlreadyExists
		}
	}

	// Assign a new UID if not set
	if mailbox.UID == 0 {
		mailbox.UID = uint32(len(db.Mailboxes) + 1)
	}

	db.Mailboxes = append(db.Mailboxes, mailbox)
	return nil
}

func (db *DB) UpdateMailbox(mailbox mails.Mailbox) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, existing := range db.Mailboxes {
		if existing.UserID == mailbox.UserID && existing.UID == mailbox.UID {
			db.Mailboxes[i] = mailbox
			return nil
		}
	}
	return mails.ErrMailboxNotFound
}

func (db *DB) DeleteMailbox(userID string, uid uint32) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, mailbox := range db.Mailboxes {
		if mailbox.UserID == userID && mailbox.UID == uid {
			db.Mailboxes = append(db.Mailboxes[:i], db.Mailboxes[i+1:]...)
			return nil
		}
	}
	return mails.ErrMailboxNotFound
}

func (db *DB) GetMails(userID string, mailboxUID uint32) ([]mails.Mail, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var userMails []mails.Mail
	for _, mail := range db.Mails {
		if mail.MailboxUID == mailboxUID {
			userMails = append(userMails, mail)
		}
	}
	return userMails, nil
}

func (db *DB) GetMailByUID(userID string, mailboxUID uint32, uid uint32) (*mails.Mail, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	mb, err := db.GetMailboxByUID(userID, mailboxUID)
	if err != nil {
		return nil, err
	}

	for _, mail := range db.Mails {
		if mail.MailboxUID == mb.UID && mail.UID == uid {
			return &mail, nil
		}
	}

	return nil, mails.ErrMailNotFound
}

func (db *DB) InsertMail(userID string, mailboxUID uint32, mail mails.Mail) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if mailbox exists
	mb, err := db.GetMailboxByUID(userID, mailboxUID)
	if err != nil {
		return err
	}

	// Check if mail already exists
	for _, existing := range db.Mails {
		if existing.MailboxUID == mb.UID && existing.UID == mail.UID {
			return mails.ErrMailAlreadyExists
		}
	}

	// Assign a new UID if not set
	if mail.UID == 0 {
		mail.UID = uint32(len(db.Mails) + 1)
	}

	mail.MailboxUID = mb.UID
	db.Mails = append(db.Mails, mail)
	return nil
}

func (db *DB) UpdateMail(userID string, mailboxUID uint32, mail mails.Mail) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	mb, err := db.GetMailboxByUID(userID, mailboxUID)
	if err != nil {
		return err
	}

	for i, existing := range db.Mails {
		if existing.MailboxUID == mb.UID && existing.UID == mail.UID {
			db.Mails[i] = mail
			return nil
		}
	}
	return mails.ErrMailNotFound
}

func (db *DB) DeleteMail(userID string, mailboxUID uint32, uid uint32) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	mb, err := db.GetMailboxByUID(userID, mailboxUID)
	if err != nil {
		return err
	}

	for i, mail := range db.Mails {
		if mail.MailboxUID == mb.UID && mail.UID == uid {
			db.Mails = append(db.Mails[:i], db.Mails[i+1:]...)
			return nil
		}
	}
	return mails.ErrMailNotFound
}
