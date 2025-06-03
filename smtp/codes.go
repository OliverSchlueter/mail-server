package smtp

const (
	StatusServiceReady = "220 %s SMTP service ready" // server hostname
	StatusConnClosed   = "221 %s closing connection" // server hostname
	StatusAuthSuccess  = "235 Authentication successful"
	StatusOK           = "250 OK"
	StatusGreeting     = "250 %s greets %s" // server hostname, client hostname

	StatusAuthUsername   = "334 VXNlcm5hbWU6" // Base64 encoded "Username:"
	StatusAuthPassword   = "334 UGFzc3dvcmQ6" // Base64 encoded "Password:"
	StatusStartMailInput = "354 Start mail input; end with <CRLF>.<CRLF>"

	StatusBadCommand    = "500 Unrecognized command"
	StatusInvalidBase64 = "501 Invalid base64 encoding"
	StatusBadSequence   = "503 Bad sequence: '%s' required first" // required command
	StatusAuthRequired  = "530 Authentication required"
	StatusNoSuchUser    = "550 No such user here"
)
