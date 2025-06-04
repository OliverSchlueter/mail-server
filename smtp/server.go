package smtp

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/OliverSchlueter/goutils/sloki"
	"log/slog"
	"net"
	"strings"
)

type Server struct {
	hostname string
	port     string
}

type Configuration struct {
	Hostname string
	Port     string
}

func NewServer(config Configuration) *Server {
	if config.Port == "" {
		config.Port = "25"
	}

	return &Server{
		hostname: config.Hostname,
		port:     config.Port,
	}
}

func (s *Server) StartServer() {
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

	slog.Debug("New connection established", "remote_addr", conn.RemoteAddr().String(), "protocol", conn.RemoteAddr().Network())

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	writeLine(w, fmt.Sprintf(StatusServiceReady, s.hostname))

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			slog.Warn("Failed to read from connection", sloki.WrapError(err))
			return
		}

		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)

		slog.Debug("C: " + line)

		if session.Mail.ReadingData {
			if line == "." {
				writeLine(w, StatusOK)
				slog.Debug(fmt.Sprintf("Email received %#v", s))
				// TODO store email

				// Reset session for next email
				session.Mail.DataBuffer = nil
				session.Mail.From = ""
				session.Mail.To = nil
				session.Mail.ReadingData = false
			} else {
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

			// TODO validate credentials
			session.AuthLogin.IsAuthenticated = true

			writeLine(w, StatusAuthSuccess)
			continue
		}

		switch {
		// EHLO
		case strings.HasPrefix(upper, CmdEhlo.Prefix):
			clientHostname := line[len(CmdEhlo.Prefix):]
			session.HeloReceived = true
			session.Hostname = clientHostname
			writeLine(w, fmt.Sprintf(StatusGreeting, s.hostname, clientHostname))

			// extensions
			writeLine(w, CmdAuthLogin.Structure)
			writeLine(w, CmdAuthPlain.Structure)

		// HELO
		case strings.HasPrefix(upper, CmdHelo.Prefix):
			clientHostname := line[len(CmdHelo.Prefix):]
			session.HeloReceived = true
			session.Hostname = clientHostname
			writeLine(w, fmt.Sprintf(StatusGreeting, s.hostname, clientHostname))

		// MAIL FROM
		case strings.HasPrefix(upper, CmdMailFrom.Prefix):
			if !session.HeloReceived {
				slog.Warn(fmt.Sprintf("%s command received before %s", CmdMailFrom.Name, CmdEhlo.Name))
				writeLine(w, fmt.Sprintf(StatusBadSequence, CmdEhlo.Name))
				continue
			}

			// TODO require authentication before MAIL FROM

			session.Mail.From = strings.TrimPrefix(line, CmdMailFrom.Prefix)
			session.Mail.From = strings.Trim(session.Mail.From, "<> ")
			writeLine(w, StatusOK)

		// AUTH LOGIN
		case upper == CmdAuthLogin.Prefix:
			if !session.HeloReceived {
				slog.Warn(fmt.Sprintf("%s command received before %s", CmdAuthLogin.Name, CmdEhlo.Name))
				writeLine(w, fmt.Sprintf(StatusBadSequence, CmdEhlo.Name))
				continue
			}

			session.AuthLogin.RequestedUsername = true
			writeLine(w, StatusAuthUsername) // Request username

		// AUTH PLAIN
		case upper == CmdAuthPlain.Prefix:
			if !session.HeloReceived {
				slog.Warn(fmt.Sprintf("%s command received before %s", CmdAuthPlain.Name, CmdEhlo.Name))
				writeLine(w, fmt.Sprintf(StatusBadSequence, CmdEhlo.Name))
				continue
			}

			credentials := strings.TrimPrefix(line, CmdAuthPlain.Prefix)

			decoded, err := base64.StdEncoding.DecodeString(credentials)
			if err != nil {
				slog.Warn("Failed to decode base64 credentials", sloki.WrapError(err))
				writeLine(w, StatusInvalidBase64)
				continue
			}

			parts := strings.SplitN(string(decoded), "\x00", 3)
			if len(parts) != 3 {
				slog.Warn("Invalid AUTH PLAIN credentials format")
				writeLine(w, StatusInvalidBase64)
				continue
			}

			session.AuthLogin.Username = parts[1]
			session.AuthLogin.Password = parts[2]

			// TODO validate credentials
			session.AuthLogin.IsAuthenticated = true
			writeLine(w, StatusAuthSuccess)

		// RCPT TO
		case strings.HasPrefix(upper, CmdRcptTo.Prefix):
			if !session.HeloReceived {
				slog.Warn(fmt.Sprintf("%s command received before %s", CmdRcptTo.Name, CmdMailFrom.Name))
				writeLine(w, fmt.Sprintf(StatusBadSequence, CmdMailFrom.Name))
				continue
			}

			recipient := strings.TrimPrefix(line, CmdRcptTo.Prefix)
			recipient = strings.Trim(recipient, "<> ")
			//TODO check if recipient exists ("550 No such user here"
			session.Mail.To = append(session.Mail.To, recipient)
			writeLine(w, StatusOK)

		// DATA
		case upper == CmdData.Prefix:
			if len(session.Mail.To) == 0 {
				slog.Warn(fmt.Sprintf("%s command received without any recipients", CmdData.Name))
				writeLine(w, fmt.Sprintf(StatusBadSequence, CmdRcptTo.Name))
				continue
			}
			session.Mail.ReadingData = true
			writeLine(w, StatusStartMailInput)

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
