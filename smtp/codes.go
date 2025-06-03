package smtp

const (
	StatusOK             = "250 OK"
	StatusGreeting       = "250 %s greets %s"          // server hostname, client hostname
	StatusConnClosed     = "221 %s closing connection" // server hostname
	StatusStartMailInput = "354 Start mail input; end with <CRLF>.<CRLF>"
	StatusBadCommand     = "500 Unrecognized command"
	StatusBadSequence    = "503 Bad sequence: '%s' required first" // required command
	StatusNoSuchUser     = "550 No such user here"
)
