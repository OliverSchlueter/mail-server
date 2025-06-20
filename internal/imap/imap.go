package imap

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"github.com/OliverSchlueter/goutils/sloki"
	"github.com/OliverSchlueter/mail-server/internal/users"
	"log/slog"
	"net"
	"strings"
	"time"
)

type Server struct {
	port      string
	users     users.Store
	tlsConfig *tls.Config
}

type Configuration struct {
	Port     string
	Users    users.Store
	CertFile string
	KeyFile  string
}

func NewServer(config Configuration) *Server {
	if config.Port == "" {
		config.Port = "143"
	}

	var tlsConfig *tls.Config
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			slog.Error("Failed to load TLS certificates", sloki.WrapError(err))
		} else {
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		}
	}

	return &Server{
		port:      config.Port,
		users:     config.Users,
		tlsConfig: tlsConfig,
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

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	session := &Session{}
	session.RemoteAddr = conn.RemoteAddr().String()
	session.IsTLS = false

	slog.Debug("New connection established", "remote_addr", conn.RemoteAddr().String(), "protocol", conn.RemoteAddr().Network())

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	writeLine(w, "* OK IMAP4rev2 Service Ready")

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

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		slog.Debug("C: " + line)

		parts := strings.SplitN(line, " ", 2)
		tag := parts[0]
		var command string
		var args string
		if len(parts) > 1 {
			split := strings.SplitN(parts[1], " ", 2)
			command = strings.ToUpper(split[0])
			if len(split) > 1 {
				args = split[1]
			}
		} else {
			writeLine(w, tag+" BAD Missing command")
			continue
		}

		switch command {
		case "CAPABILITY":
			writeLine(w, "* CAPABILITY IMAP4rev2 STARTTLS AUTH=PLAIN UTF8=ACCEPT")
			writeLine(w, tag+" OK CAPABILITY completed")

		case "STARTTLS":
			if session.IsTLS {
				writeLine(w, tag+" BAD TLS already active")
				continue
			}
			if s.tlsConfig == nil {
				writeLine(w, tag+" BAD TLS not supported on this server")
				continue
			}

			writeLine(w, tag+" OK Begin TLS negotiation now")
			tlsConn := tls.Server(conn, s.tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				slog.Error("TLS handshake failed", sloki.WrapError(err))
				return
			}

			// Re-wrap buffered readers/writers over TLS connection
			conn = tlsConn
			r = bufio.NewReader(conn)
			w = bufio.NewWriter(conn)
			session.IsTLS = true
			writeLine(w, "* OK TLS negotiation completed")

		case "AUTHENTICATE":
			if !session.IsTLS {
				writeLine(w, tag+" BAD Must issue STARTTLS before authentication")
				continue
			}
			args = strings.TrimSpace(args)
			if args != "PLAIN" {
				writeLine(w, tag+" BAD Only AUTHENTICATE PLAIN supported currently")
				continue
			}

			writeLine(w, "+") // Challenge (empty for PLAIN)

			authLine, err := r.ReadString('\n')
			if err != nil {
				writeLine(w, tag+" NO Failed to read authentication data")
				return
			}
			authLine = strings.TrimSpace(authLine)

			decoded, err := base64.StdEncoding.DecodeString(authLine)
			if err != nil {
				writeLine(w, tag+" NO Invalid base64 in AUTHENTICATE")
				continue
			}

			username, password, err := parsePlainAuth(decoded)
			if err != nil {
				writeLine(w, tag+" NO Invalid AUTHENTICATE format")
				continue
			}

			u, err := s.users.GetByEmail(username)
			if err != nil {
				writeLine(w, tag+" NO User not found: "+username)
				continue
			}
			if u.Password != users.Hash(password) {
				writeLine(w, tag+" NO Invalid password for user: "+username)
				continue
			}
			writeLine(w, tag+" OK Authentication successful")

		case "NOOP":
			writeLine(w, tag+" OK NOOP completed")

		case "LOGOUT":
			writeLine(w, "* BYE IMAP4rev2 Server logging out")
			writeLine(w, tag+" OK LOGOUT completed")
			return

		default:
			writeLine(w, tag+" BAD Unknown or unsupported command: "+command)
		}
	}
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

func parsePlainAuth(decoded []byte) (username, password string, err error) {
	parts := strings.SplitN(string(decoded), "\x00", 3)
	if len(parts) != 3 {
		return "", "", errors.New("invalid AUTHENTICATE PLAIN format")
	}
	username = parts[1]
	password = parts[2]
	return username, password, nil
}
