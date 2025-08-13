package imap

import "github.com/OliverSchlueter/mail-server/internal/users"

type Session struct {
	RemoteAddr     string
	IsTLS          bool
	Authentication Authentication
}

type Authentication struct {
	User            *users.User
	IsAuthenticated bool
}
