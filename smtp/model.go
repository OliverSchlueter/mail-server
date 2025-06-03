package smtp

type Session struct {
	Hostname     string
	RemoteAddr   string
	HeloReceived bool
	AuthLogin    AuthLogin
	MailFrom     string
	RcptTo       []string
	DataBuffer   []string
	ReadingData  bool
}

type AuthLogin struct {
	RequestedUsername bool
	Username          string
	RequestedPassword bool
	Password          string
	IsAuthenticated   bool
}
