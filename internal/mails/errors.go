package mails

import "errors"

var (
	ErrMailboxNotFound      = errors.New("mailbox not found")
	ErrMailboxAlreadyExists = errors.New("mailbox already exists")
	ErrMailNotFound         = errors.New("mail not found")
	ErrMailAlreadyExists    = errors.New("mail already exists")
)
