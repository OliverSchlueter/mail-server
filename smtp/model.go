package smtp

type Session struct {
	HeloReceived bool
	MailFrom     string
	RcptTo       []string
	DataBuffer   []string
	ReadingData  bool
}
