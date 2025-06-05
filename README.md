# Mail server

Simple mail server written in Go. Everything is written from scratch, no external dependencies.

It is designed to be a lightweight and easy-to-use solution.

## Features

Supported protocols:
- SMTP (incoming and outgoing)
- IMAP (work in progress)

Planned protocols:
- Calendar (CalDAV)
- Contacts (CardDAV)
- LDAP
- OAuth2/OIDC
- DNS

Planned other features:
- Frontend: webclient, mobile app, desktop app
- REST API (for all services)
- Webhooks for events (e.g. new email, new calendar event, etc.)
- Discord notifications
- NATS integration for blazingly fast event processing
- AI integration for smart features (e.g. email classification, spam detection, etc.)
