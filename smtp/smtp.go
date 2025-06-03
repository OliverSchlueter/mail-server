package smtp

import (
	"bufio"
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

		if session.ReadingData {
			if line == "." {
				writeLine(w, StatusOK)
				slog.Debug(fmt.Sprintf("Email received %#v", s))
				// TODO store email

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
		case strings.HasPrefix(upper, CmdEhlo.Prefix):
			clientHostname := line[len(CmdEhlo.Prefix):]
			session.HeloReceived = true
			session.Hostname = clientHostname
			writeLine(w, fmt.Sprintf(StatusGreeting, s.hostname, clientHostname))
			//TODO send supported extensions

		case strings.HasPrefix(upper, CmdMailFrom.Prefix):
			if !session.HeloReceived {
				slog.Warn(fmt.Sprintf("%s command received before %s", CmdMailFrom.Name, CmdEhlo.Name))
				writeLine(w, fmt.Sprintf(StatusBadSequence, CmdEhlo.Name))
				continue
			}
			session.MailFrom = strings.TrimPrefix(line, CmdMailFrom.Prefix)
			session.MailFrom = strings.Trim(session.MailFrom, "<> ")
			writeLine(w, StatusOK)

		case strings.HasPrefix(upper, CmdRcptTo.Prefix):
			if !session.HeloReceived {
				slog.Warn(fmt.Sprintf("%s command received before %s", CmdRcptTo.Name, CmdMailFrom.Name))
				writeLine(w, fmt.Sprintf(StatusBadSequence, CmdMailFrom.Name))
				continue
			}

			recipient := strings.TrimPrefix(line, CmdRcptTo.Prefix)
			recipient = strings.Trim(recipient, "<> ")
			//TODO check if recipient exists ("550 No such user here"
			session.RcptTo = append(session.RcptTo, recipient)
			writeLine(w, StatusOK)

		case upper == CmdData.Prefix:
			if len(session.RcptTo) == 0 {
				slog.Warn(fmt.Sprintf("%s command received without any recipients", CmdData.Name))
				writeLine(w, fmt.Sprintf(StatusBadSequence, CmdRcptTo.Name))
				continue
			}
			session.ReadingData = true
			writeLine(w, StatusStartMailInput)

		case upper == CmdRset.Prefix:
			session = &Session{} // Reset session
			writeLine(w, StatusOK)

		case upper == CmdQuit.Prefix:
			writeLine(w, fmt.Sprintf(StatusConnClosed, s.hostname))
			slog.Debug("Connection closed", "remote_addr", session.RemoteAddr)
			return

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
