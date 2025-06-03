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

	// extensions
	CmdAuthLogin = Command{
		Name:      "AUTH LOGIN",
		Prefix:    "AUTH LOGIN",
		Structure: "250-AUTH LOGIN",
	}
)
