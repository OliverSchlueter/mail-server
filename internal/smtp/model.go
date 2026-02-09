package smtp

const (
	MaxMessageSize = 15 * 1024 * 1024 // 15 MB
	MaxRecipients  = 100
)

type Session struct {
	Hostname     string
	RemoteAddr   string
	TLSActive    bool
	HeloReceived bool
	Mail         Mail
	AuthLogin    AuthLogin // state for AUTH LOGIN authentication flow
	DeliveryUser string    // the user that should receive the mail, determined by the RCPT TO command
}

type Mail struct {
	Outgoing    bool
	From        string
	To          []string
	DataBuffer  []string
	Subject     string
	Domain      string
	ReadingData bool
}

func (m *Mail) Size() int {
	size := 0
	for _, line := range m.DataBuffer {
		size += len(line) + 1 // +1 for the newline character
	}
	return size
}

func (m *Mail) Body() string {
	body := ""
	for _, line := range m.DataBuffer {
		body += line + "\n"
	}
	return body
}

type AuthLogin struct {
	RequestedUsername bool
	Username          string
	RequestedPassword bool
	Password          string
	IsAuthenticated   bool
}
