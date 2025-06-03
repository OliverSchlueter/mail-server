package smtp

import (
	"bufio"
	"fmt"
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
			slog.Warn("Failed to accept connection", "error", err)
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

	writeLine(w, fmt.Sprintf("220 %s Mail Transfer Service Ready", s.hostname))

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			slog.Warn("Failed to read from connection", "error", err)
			return
		}

		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)

		slog.Debug("C: " + line)

		if session.ReadingData {
			if line == "." {
				writeLine(w, StatusOK)
				slog.Debug(fmt.Sprintf("Email received %#v", s))

				// Reset session for next email
				session.DataBuffer = nil
				session.MailFrom = ""
				session.RcptTo = nil
				session.ReadingData = false
			} else {
				session.DataBuffer = append(session.DataBuffer, line)
			}

			continue
		}

		switch {
		case strings.HasPrefix(upper, "EHLO"):
			clientHostname := line[len("EHLO "):]
			session.HeloReceived = true
			session.Hostname = clientHostname
			writeLine(w, fmt.Sprintf(StatusGreeting, s.hostname, clientHostname))

		case strings.HasPrefix(upper, "MAIL FROM:"):
			if !session.HeloReceived {
				writeLine(w, fmt.Sprintf(StatusBadSequence, "EHLO"))
				continue
			}
			session.MailFrom = strings.TrimPrefix(line, "MAIL FROM:")
			session.MailFrom = strings.Trim(session.MailFrom, "<> ")
			writeLine(w, StatusOK)

		case strings.HasPrefix(upper, "RCPT TO:"):
			if !session.HeloReceived {
				writeLine(w, fmt.Sprintf(StatusBadSequence, "MAIL FROM"))
				continue
			}

			recipient := strings.TrimPrefix(line, "RCPT TO:")
			recipient = strings.Trim(recipient, "<> ")
			//TODO check if recipient exists ("550 No such user here"
			session.RcptTo = append(session.RcptTo, recipient)
			writeLine(w, StatusOK)

		case upper == "DATA":
			if len(session.RcptTo) == 0 {
				writeLine(w, fmt.Sprintf(StatusBadSequence, "RCPT TO"))
				continue
			}
			session.ReadingData = true
			writeLine(w, StatusStartMailInput)

		case upper == "RSET":
			session = &Session{} // Reset session
			writeLine(w, StatusOK)

		case upper == "QUIT":
			writeLine(w, fmt.Sprintf(StatusConnClosed, s.hostname))
			return

		default:
			writeLine(w, StatusBadCommand)
		}
	}
}

func writeLine(w *bufio.Writer, line string) {
	w.WriteString(line + "\n")
	w.Flush()
	slog.Debug("S: " + line)
}
