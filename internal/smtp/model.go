package smtp

type Session struct {
	Hostname     string
	RemoteAddr   string
	HeloReceived bool
	Mail         Mail
	AuthLogin    AuthLogin
}

type Mail struct {
	From        string
	To          []string
	DataBuffer  []string
	ReadingData bool
}

type AuthLogin struct {
	RequestedUsername bool
	Username          string
	RequestedPassword bool
	Password          string
	IsAuthenticated   bool
}
