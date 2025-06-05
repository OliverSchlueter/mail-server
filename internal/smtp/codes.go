package smtp

const (
	StatusServiceReady  = "220 %s SMTP service ready" // server hostname
	StatusReadyStarting = "220 Ready to start TLS"    // server hostname
	StatusConnClosed    = "221 %s closing connection" // server hostname
	StatusAuthSuccess   = "235 Authentication successful"
	StatusOK            = "250 OK"
	StatusGreeting      = "250-%s greets %s" // server hostname, client hostname

	StatusAuthUsername   = "334 VXNlcm5hbWU6" // Base64 encoded "Username:"
	StatusAuthPassword   = "334 UGFzc3dvcmQ6" // Base64 encoded "Password:"
	StatusStartMailInput = "354 Start mail input; end with <CRLF>.<CRLF>"

	StatusBadCommand           = "500 Unrecognized command"
	StatusLineTooLong          = "500 Line too long" // line exceeds maximum length
	StatusInvalidBase64        = "501 Invalid base64 encoding"
	StatusNotImplemented       = "502 Command not implemented"           // command not supported by server
	StatusBadSequence          = "503 Bad sequence: '%s' required first" // required command
	StatusAuthRequired         = "530 Authentication required"
	StatusAuthenticationFailed = "535 Authentication failed" // invalid credentials
	StatusEncryptionRequired   = "538 Encryption required for requested authentication mechanism"
	StatusNoSuchUser           = "550 No such user here"
)
