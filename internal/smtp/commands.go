package smtp

type Command struct {
	Name      string
	Structure string
	Prefix    string
}

var (
	CmdEhlo = Command{
		Name:      "EHLO",
		Prefix:    "EHLO ",
		Structure: "EHLO-%s",
	}

	CmdHelo = Command{
		Name:      "HELO",
		Prefix:    "HELO ",
		Structure: "HELO %s",
	}

	CmdStartTls = Command{
		Name:      "STARTTLS",
		Prefix:    "STARTTLS",
		Structure: "250-STARTTLS",
	}

	CmdMailFrom = Command{
		Name:      "MAIL FROM",
		Prefix:    "MAIL FROM:",
		Structure: "MAIL FROM:<%s>",
	}

	CmdRcptTo = Command{
		Name:      "RCPT TO",
		Prefix:    "RCPT TO:",
		Structure: "RCPT TO:<%s>",
	}

	CmdData = Command{
		Name:      "DATA",
		Prefix:    "DATA",
		Structure: "DATA",
	}

	CmdRset = Command{
		Name:      "RSET",
		Prefix:    "RSET",
		Structure: "RSET",
	}

	CmdQuit = Command{
		Name:      "QUIT",
		Prefix:    "QUIT",
		Structure: "QUIT",
	}

	CmdNoop = Command{
		Name:      "NOOP",
		Prefix:    "NOOP",
		Structure: "NOOP",
	}

	// extensions

	CmdAuthLogin = Command{
		Name:      "AUTH LOGIN",
		Prefix:    "AUTH LOGIN ",
		Structure: "250-AUTH LOGIN",
	}

	CmdAuthPlain = Command{
		Name:      "AUTH PLAIN",
		Prefix:    "AUTH PLAIN ",
		Structure: "250-AUTH PLAIN",
	}
)
