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

	slog.Info("New connection established", "remote_addr", conn.RemoteAddr().String(), "protocol", conn.RemoteAddr().Network())

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

		slog.Info("C: " + line)

		if session.ReadingData {
			if line == "." {
				writeLine(w, "250 OK")
				slog.Info(fmt.Sprintf("Email received %#v", s))

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
			session.HeloReceived = true
			writeLine(w, fmt.Sprintf("250-%s greets %s", s.hostname, line[len("EHLO "):]))

		case strings.HasPrefix(upper, "MAIL FROM:"):
			if !session.HeloReceived {
				writeLine(w, "503 Bad sequence: HELO required first")
				continue
			}
			session.MailFrom = strings.TrimPrefix(line, "MAIL FROM:")
			session.MailFrom = strings.Trim(session.MailFrom, "<> ")
			writeLine(w, "250 OK")

		case strings.HasPrefix(upper, "RCPT TO:"):
			if !session.HeloReceived {
				writeLine(w, "503 Bad sequence: MAIL FROM required first")
				continue
			}

			recipient := strings.TrimPrefix(line, "RCPT TO:")
			recipient = strings.Trim(recipient, "<> ")
			//TODO check if recipient exists ("550 No such user here"
			session.RcptTo = append(session.RcptTo, recipient)
			writeLine(w, "250 OK")

		case upper == "DATA":
			if len(session.RcptTo) == 0 {
				writeLine(w, "503 Bad sequence: RCPT TO required before DATA")
				continue
			}
			session.ReadingData = true
			writeLine(w, "354 Start mail input; end with <CRLF>.<CRLF>")

		case upper == "RSET":
			session = &Session{} // Reset session
			writeLine(w, "250 OK")

		case upper == "QUIT":
			writeLine(w, fmt.Sprintf("221 %s Service closing transmission channel", s.hostname))
			return

		default:
			writeLine(w, "500 Unrecognized command")
		}
	}
}

func writeLine(w *bufio.Writer, line string) {
	w.WriteString(line + "\n")
	w.Flush()
	slog.Info("S: " + line)
}
