package smtp

type Session struct {
	Hostname     string
	RemoteAddr   string
	HeloReceived bool
	MailFrom     string
	RcptTo       []string
	DataBuffer   []string
	ReadingData  bool
}
