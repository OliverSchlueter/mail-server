package smtp

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/OliverSchlueter/goutils/sloki"
	"github.com/OliverSchlueter/mail-server/internal/mails"
	"github.com/OliverSchlueter/mail-server/internal/users"
)

type Server struct {
	hostname  string
	port      string
	tlsConfig *tls.Config
	users     users.Store
	mails     mails.Store
}

type Configuration struct {
	Hostname string
	Port     string
	CertFile string
	KeyFile  string
	Users    users.Store
	Mails    mails.Store
}

func NewServer(config Configuration) *Server {
	if config.Port == "" {
		config.Port = "25"
	}

	var tlsConfig *tls.Config
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			slog.Error("Failed to load TLS certificates", sloki.WrapError(err))
		} else {
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS13,
				CipherSuites: []uint16{
					tls.TLS_AES_128_GCM_SHA256,
					tls.TLS_AES_256_GCM_SHA384,
					tls.TLS_CHACHA20_POLY1305_SHA256,
				},
				SessionTicketsDisabled: true,
				Renegotiation:          tls.RenegotiateNever,
				CurvePreferences:       []tls.CurveID{tls.X25519, tls.CurveP256},
			}
		}
	}

	return &Server{
		hostname:  config.Hostname,
		port:      config.Port,
		tlsConfig: tlsConfig,
		users:     config.Users,
		mails:     config.Mails,
	}
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Warn("Failed to accept connection", sloki.WrapError(err))
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) StartWithTLS() {
	listener, err := tls.Listen("tcp", ":465", s.tlsConfig)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Warn("Failed to accept connection", sloki.WrapError(err))
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	session := &Session{}
	session.RemoteAddr = conn.RemoteAddr().String()

	slog.Debug("New connection established", "remote_addr", conn.RemoteAddr().String(), "protocol", conn.RemoteAddr().Network())

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	writeLine(w, fmt.Sprintf(StatusServiceReady, s.hostname))

	for {
		if err := conn.SetDeadline(time.Now().Add(time.Duration(2) * time.Minute)); err != nil {
			slog.Error("Failed to set connection deadline", sloki.WrapError(err))
			return
		}

		line, err := r.ReadString('\n')
		if err != nil {
			slog.Warn("Failed to read from connection", sloki.WrapError(err))
			return
		}

		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)

		if len(line) > 1000 {
			slog.Warn("Received line exceeds maximum length", "line_length", len(line))
			writeLine(w, StatusLineTooLong)
			continue
		}

		slog.Debug("C: " + line)

		if session.Mail.ReadingData {
			if line == "." {
				// Defensive: if an outgoing flag somehow exists, reject relaying.
				if session.Mail.Outgoing {
					slog.Warn("Relaying attempted during DATA but relay support is disabled")
					writeLine(w, StatusRelayDenied)
				} else {
					m := mails.Mail{
						UID:        mails.RandomUID(),
						MailboxUID: mails.DefaultMailboxUID,
						Flags:      []string{},
						Date:       time.Now(),
						Size:       len(session.Mail.Body()),
						Headers:    session.Mail.Headers(),
						Body:       session.Mail.Body(),
					}
					err := s.mails.CreateMail(session.DeliveryUser, mails.DefaultMailboxUID, m)
					if err != nil {
						slog.Error("Failed to save incoming email", sloki.WrapError(err))
						writeLine(w, StatusInternalServerError)

						// reset reading state and continue
						session.Mail.DataBuffer = nil
						session.Mail.From = ""
						session.Mail.To = nil
						session.Mail.ReadingData = false
						continue
					}
					slog.Info("Incoming email received", "mail", session.Mail)
					writeLine(w, StatusOK)
				}

				// Reset session for next email
				session.Mail.DataBuffer = nil
				session.Mail.From = ""
				session.Mail.To = nil
				session.Mail.ReadingData = false
			} else {
				if strings.HasPrefix(line, ".") {
					line = line[1:]
				}

				if session.Mail.Size() > MaxMessageSize {
					writeLine(w, StatusMessageTooLarge)
					return
				}

				session.Mail.DataBuffer = append(session.Mail.DataBuffer, line)
			}

			continue
		} else if session.AuthLogin.RequestedUsername {
			decoded, err := base64.StdEncoding.DecodeString(line)
			if err != nil {
				slog.Warn("Failed to decode base64 username", sloki.WrapError(err))
				writeLine(w, StatusInvalidBase64)
				continue
			}

			session.AuthLogin.Username = string(decoded)
			session.AuthLogin.RequestedUsername = false
			session.AuthLogin.RequestedPassword = true
			writeLine(w, StatusAuthPassword) // Request password
		} else if session.AuthLogin.RequestedPassword {
			decoded, err := base64.StdEncoding.DecodeString(line)
			if err != nil {
				slog.Warn("Failed to decode base64 password", sloki.WrapError(err))
				writeLine(w, StatusInvalidBase64)
				continue
			}

			session.AuthLogin.Password = string(decoded)
			session.AuthLogin.RequestedPassword = false

			u, err := s.users.GetByName(session.AuthLogin.Username)
			if err != nil {
				if errors.Is(err, users.ErrUserNotFound) {
					writeLine(w, StatusNoSuchUser)
					continue
				}

				slog.Error("Failed to get user by name", sloki.WrapError(err))
				continue
			}

			if u.Password != users.Hash(session.AuthLogin.Password) {
				writeLine(w, StatusAuthenticationFailed)
				continue
			}

			session.AuthLogin.IsAuthenticated = true

			writeLine(w, StatusAuthSuccess)
			continue
		}

		switch {
		// EHLO
		case strings.HasPrefix(upper, CmdEhlo.Prefix):
			s.handlEhlo(session, w, line)

		// HELO
		case strings.HasPrefix(upper, CmdHelo.Prefix):
			s.handleHelo(session, w, line)

		// STARTTLS
		case upper == CmdStartTls.Prefix:
			if s.tlsConfig == nil {
				writeLine(w, StatusNotImplemented)
				return
			}

			if session.TLSActive {
				writeLine(w, StatusNotImplemented)
				return
			}

			writeLine(w, StatusReadyStarting)

			// Upgrade connection to TLS
			tlsConn := tls.Server(conn, s.tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				slog.Error("TLS handshake failed", sloki.WrapError(err))
				return
			}

			// Reset session but keep remote address
			remoteAddr := session.RemoteAddr
			*session = Session{RemoteAddr: remoteAddr, TLSActive: true}

			// Update connection and readers/writers
			conn = tlsConn
			r = bufio.NewReader(conn)
			w = bufio.NewWriter(conn)

			slog.Debug("TLS connection established", "remote_addr", conn.RemoteAddr().String())

		// AUTH LOGIN
		case upper == CmdAuthLogin.Prefix:
			s.handleAuthLogin(session, w, line)

		// AUTH PLAIN
		case strings.HasPrefix(upper, CmdAuthPlain.Prefix):
			s.handleAuthPlain(session, w, line)

		// MAIL FROM
		case strings.HasPrefix(upper, CmdMailFrom.Prefix):
			s.handleMailFrom(session, w, line)

		// RCPT TO
		case strings.HasPrefix(upper, CmdRcptTo.Prefix):
			s.handleRcptTo(session, w, line)

		// DATA
		case upper == CmdData.Prefix:
			s.handleData(session, w, line)

		// RSET
		case upper == CmdRset.Prefix:
			session = &Session{} // Reset session
			writeLine(w, StatusOK)

		// QUIT
		case upper == CmdQuit.Prefix:
			writeLine(w, fmt.Sprintf(StatusConnClosed, s.hostname))
			slog.Debug("Connection closed", "remote_addr", session.RemoteAddr)
			return

		// NOOP
		case upper == CmdNoop.Prefix:
			writeLine(w, StatusOK)

		default:
			writeLine(w, StatusBadCommand)
		}
	}
}

func (s *Server) handlEhlo(session *Session, w *bufio.Writer, line string) {
	clientHostname := line[len(CmdEhlo.Prefix):]
	session.HeloReceived = true
	session.Hostname = clientHostname
	writeLine(w, fmt.Sprintf(StatusGreeting, s.hostname, clientHostname))

	if !session.TLSActive && s.tlsConfig != nil {
		writeLine(w, CmdStartTls.Structure)
		return
	}

	if session.TLSActive {
		writeLine(w, CmdAuthLogin.Structure)
		writeLine(w, strings.Replace(CmdAuthPlain.Structure, "-", " ", 1))
	}
}

func (s *Server) handleHelo(session *Session, w *bufio.Writer, line string) {
	clientHostname := line[len(CmdHelo.Prefix):]
	session.HeloReceived = true
	session.Hostname = clientHostname

	writeLine(w, fmt.Sprintf(StatusGreeting, s.hostname, clientHostname))
}

func (s *Server) handleAuthLogin(session *Session, w *bufio.Writer, line string) {
	if !session.HeloReceived {
		slog.Warn(fmt.Sprintf("%s command received before %s", CmdAuthLogin.Name, CmdEhlo.Name))
		writeLine(w, fmt.Sprintf(StatusBadSequence, CmdEhlo.Name))
		return
	}

	if !session.TLSActive {
		writeLine(w, StatusEncryptionRequired)
		return
	}

	session.AuthLogin.RequestedUsername = true
	writeLine(w, StatusAuthUsername) // Request username
}

func (s *Server) handleAuthPlain(session *Session, w *bufio.Writer, line string) {
	if !session.HeloReceived {
		slog.Warn(fmt.Sprintf("%s command received before %s", CmdAuthPlain.Name, CmdEhlo.Name))
		writeLine(w, fmt.Sprintf(StatusBadSequence, CmdEhlo.Name))
		return
	}

	if !session.TLSActive {
		writeLine(w, StatusEncryptionRequired)
		return
	}

	credentials := strings.TrimPrefix(line, CmdAuthPlain.Prefix)

	decoded, err := base64.StdEncoding.DecodeString(credentials)
	if err != nil {
		slog.Warn("Failed to decode base64 credentials", sloki.WrapError(err))
		writeLine(w, StatusInvalidBase64)
		return
	}

	parts := strings.SplitN(string(decoded), "\x00", 3)
	if len(parts) != 3 {
		slog.Warn("Invalid AUTH PLAIN credentials format")
		writeLine(w, StatusInvalidBase64)
		return
	}

	session.AuthLogin.Username = parts[1]
	session.AuthLogin.Password = parts[2]

	u, err := s.users.GetByName(session.AuthLogin.Username)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			writeLine(w, StatusNoSuchUser)
			return
		}

		slog.Error("Failed to get user by name", sloki.WrapError(err))
		return
	}

	if u.Password != users.Hash(session.AuthLogin.Password) {
		writeLine(w, StatusAuthenticationFailed)
		return
	}

	session.AuthLogin.IsAuthenticated = true
	writeLine(w, StatusAuthSuccess)
}

func (s *Server) handleMailFrom(session *Session, w *bufio.Writer, line string) {
	if !session.HeloReceived {
		slog.Warn(fmt.Sprintf("%s command received before %s", CmdMailFrom.Name, CmdEhlo.Name))
		writeLine(w, fmt.Sprintf(StatusBadSequence, CmdEhlo.Name))
		return
	}

	if s.tlsConfig != nil && !session.TLSActive {
		slog.Warn(fmt.Sprintf("%s command received without TLS", CmdMailFrom.Name))
		writeLine(w, StatusEncryptionRequired)
		return
	}

	if session.TLSActive && !session.AuthLogin.IsAuthenticated {
		writeLine(w, StatusAuthRequired)
		return
	}

	addr := strings.TrimPrefix(line, CmdMailFrom.Prefix)
	addr = strings.TrimSpace(strings.Trim(addr, "<>"))

	// Allow null sender (bounce/DSN) indicated by empty address
	if addr == "" {
		session.Mail.From = ""
		session.Mail.Outgoing = false
		writeLine(w, StatusOK)
		return
	}

	parts := strings.Split(addr, "@")
	if len(parts) != 2 {
		slog.Warn(fmt.Sprintf("Invalid MAIL FROM address: %s", addr))
		writeLine(w, StatusNoSuchUser)
		return
	}

	domain := parts[1]
	// Reject relaying: only accept senders from the server's hostname (local)
	if domain != s.hostname {
		slog.Warn(fmt.Sprintf("Relaying attempt denied for MAIL FROM: %s", addr))
		writeLine(w, StatusRelayDenied)
		return
	}

	// Accept local sender
	session.Mail.From = addr
	session.Mail.Outgoing = false

	session.Mail.To = nil
	session.Mail.ReadingData = false

	writeLine(w, StatusOK)
}

func (s *Server) handleRcptTo(session *Session, w *bufio.Writer, line string) {
	if !session.HeloReceived {
		slog.Warn(fmt.Sprintf("%s command received before %s", CmdRcptTo.Name, CmdMailFrom.Name))
		writeLine(w, fmt.Sprintf(StatusBadSequence, CmdMailFrom.Name))
		return
	}

	if session.Mail.From == "" {
		writeLine(w, fmt.Sprintf(StatusBadSequence, CmdMailFrom.Name))
		return
	}

	if len(session.Mail.To) >= MaxRecipients {
		slog.Warn(fmt.Sprintf("Maximum recipients exceeded for session from %s", session.RemoteAddr))
		writeLine(w, StatusTooManyRecipients)
		return
	}

	recipient := strings.TrimPrefix(line, CmdRcptTo.Prefix)
	recipient = strings.Trim(recipient, "<> ")

	// check if it's an incoming email
	if !session.Mail.Outgoing {
		u, err := s.users.GetByEmail(recipient)
		if err != nil {
			if errors.Is(err, users.ErrUserNotFound) {
				writeLine(w, StatusNoSuchUser)
				return
			}

			slog.Error("Failed to get user by name", sloki.WrapError(err))
			return
		}
		session.DeliveryUser = u.ID
	}

	session.Mail.To = append(session.Mail.To, recipient)
	writeLine(w, StatusOK)
}

func (s *Server) handleData(session *Session, w *bufio.Writer, line string) {
	if !session.HeloReceived {
		writeLine(w, fmt.Sprintf(StatusBadSequence, CmdEhlo.Name))
		return
	}

	if len(session.Mail.To) == 0 {
		slog.Warn(fmt.Sprintf("%s command received without any recipients", CmdData.Name))
		writeLine(w, fmt.Sprintf(StatusBadSequence, CmdRcptTo.Name))
		return
	}

	session.Mail.ReadingData = true
	writeLine(w, StatusStartMailInput)
}

func writeLine(w *bufio.Writer, line string) {
	if _, err := w.WriteString(line + "\r\n"); err != nil {
		slog.Error("Failed to write to connection", sloki.WrapError(err))
		return
	}
	if err := w.Flush(); err != nil {
		slog.Error("Failed to flush writer", sloki.WrapError(err))
		return
	}

	slog.Debug("S: " + line)
}
