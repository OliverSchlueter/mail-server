package mails

import (
	"errors"
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
	mb, err := s.db.GetMailboxByUID(userID, uid)
	if err != nil {
		if errors.Is(err, ErrMailboxNotFound) && uid == DefaultMailboxUID {
			// If the default mailbox is not found, create it
			mb = &Mailbox{
				UserID: userID,
				Name:   DefaultMailboxName,
				UID:    DefaultMailboxUID,
				Flags:  []string{},
			}
			err = s.db.InsertMailbox(*mb)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return mb, nil
}

func (s *Store) GetMailboxByName(userID string, name string) (*Mailbox, error) {
	mb, err := s.db.GetMailboxByName(userID, name)
	if err != nil {
		if errors.Is(err, ErrMailboxNotFound) && name == DefaultMailboxName {
			// If the default mailbox is not found, create it
			mb = &Mailbox{
				UserID: userID,
				Name:   DefaultMailboxName,
				UID:    DefaultMailboxUID,
				Flags:  []string{},
			}
			err = s.db.InsertMailbox(*mb)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return mb, nil
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
	_, err := s.GetMailboxByUID(userID, mailboxUID)
	if err != nil {
		return ErrMailboxNotFound
	}

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
