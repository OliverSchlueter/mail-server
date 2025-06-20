package smtp

type Session struct {
	Hostname     string
	RemoteAddr   string
	TLSActive    bool
	HeloReceived bool
	Mail         Mail
	AuthLogin    AuthLogin
}

type Mail struct {
	Outgoing    bool
	From        string
	To          []string
	DataBuffer  []string
	ReadingData bool
}

func (m *Mail) Body() string {
	body := ""
	for _, line := range m.DataBuffer {
		body += line + "\n"
	}
	return body
}

func (m *Mail) Headers() map[string]string {
	// TODO: Implement proper header parsing
	return map[string]string{}
}

type AuthLogin struct {
	RequestedUsername bool
	Username          string
	RequestedPassword bool
	Password          string
	IsAuthenticated   bool
}
