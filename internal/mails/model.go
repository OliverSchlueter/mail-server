package mails

import "time"

const DefaultMailboxName = "INBOX"
const DefaultMailboxUID uint32 = 1

type Mailbox struct {
	UserID string   `json:"user_id"`
	Name   string   `json:"name"`
	UID    uint32   `json:"uid"`
	Flags  []string `json:"flags"`
}

type Mail struct {
	UID        uint32            `json:"uid"`
	MailboxUID uint32            `json:"mailbox_uid"`
	Flags      []string          `json:"flags"`
	Date       time.Time         `json:"date"`
	Size       int               `json:"size"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}
