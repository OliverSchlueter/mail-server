package mails

import (
	"math/rand"
)

type DB interface {
	GetMailboxes(userID string) ([]Mailbox, error)
	GetMailboxByUID(userID string, uid uint32) (*Mailbox, error)
	GetMailboxByName(userID string, name string) (*Mailbox, error)
	InsertMailbox(mailbox Mailbox) error
	UpdateMailbox(mailbox Mailbox) error
	DeleteMailbox(userID string, uid uint32) error

	GetMails(userID string, mailboxUID uint32) ([]Mail, error)
	GetMailByUID(userID string, mailboxUID uint32, uid uint32) (*Mail, error)
	InsertMail(userID string, mailboxUID uint32, mail Mail) error
	UpdateMail(userID string, mailboxUID uint32, mail Mail) error
	DeleteMail(userID string, mailboxUID uint32, uid uint32) error
}

type Store struct {
	db DB
}

type Configuration struct {
	DB DB
}

func NewStore(cfg Configuration) *Store {
	return &Store{
		db: cfg.DB,
	}
}

func (s *Store) GetMailboxes(userID string) ([]Mailbox, error) {
	return s.db.GetMailboxes(userID)
}

func (s *Store) GetMailboxByUID(userID string, uid uint32) (*Mailbox, error) {
	return s.db.GetMailboxByUID(userID, uid)
}

func (s *Store) GetMailboxByName(userID string, name string) (*Mailbox, error) {
	return s.db.GetMailboxByName(userID, name)
}

func (s *Store) CreateMailbox(mailbox Mailbox) error {
	return s.db.InsertMailbox(mailbox)
}

func (s *Store) UpdateMailbox(mailbox Mailbox) error {
	return s.db.UpdateMailbox(mailbox)
}

func (s *Store) DeleteMailbox(userID string, uid uint32) error {
	return s.db.DeleteMailbox(userID, uid)
}

func (s *Store) GetMails(userID string, mailboxUID uint32) ([]Mail, error) {
	return s.db.GetMails(userID, mailboxUID)
}

func (s *Store) GetMailByUID(userID string, mailboxUID uint32, uid uint32) (*Mail, error) {
	return s.db.GetMailByUID(userID, mailboxUID, uid)
}

func (s *Store) CreateMail(userID string, mailboxUID uint32, mail Mail) error {
	return s.db.InsertMail(userID, mailboxUID, mail)
}

func (s *Store) UpdateMail(userID string, mailboxUID uint32, mail Mail) error {
	return s.db.UpdateMail(userID, mailboxUID, mail)
}

func (s *Store) DeleteMail(userID string, mailboxUID uint32, uid uint32) error {
	return s.db.DeleteMail(userID, mailboxUID, uid)
}

func RandomUID() uint32 {
	return rand.Uint32()
}
