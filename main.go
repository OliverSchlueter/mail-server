package main

import (
	"bufio"
	"fmt"
	"github.com/OliverSchlueter/goutils/sloki"
	"log/slog"
	"net"
	"strings"
)

const hostname = "foo.com"

type Session struct {
	HeloReceived bool
	MailFrom     string
	RcptTo       []string
	DataBuffer   []string
	ReadingData  bool
}

func main() {
	lokiService := sloki.NewService(sloki.Configuration{
		URL:          "http://localhost:3100/loki/api/v1/push",
		Service:      "mail-server",
		ConsoleLevel: slog.LevelDebug,
		LokiLevel:    slog.LevelInfo,
		EnableLoki:   false,
	})
	slog.SetDefault(slog.New(lokiService))

	go startServer()
	slog.Info("Server started, listening on port 2525")

	c := make(chan struct{})
	<-c
}

func startServer() {
	listener, err := net.Listen("tcp", ":2525")
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

		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	s := &Session{}

	slog.Info("New connection established", "remote_addr", conn.RemoteAddr().String(), "protocol", conn.RemoteAddr().Network())

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	writeLine(w, fmt.Sprintf("220 %s Mail Transfer Service Ready", hostname))

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			slog.Warn("Failed to read from connection", "error", err)
			return
		}

		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)

		slog.Info("C: " + line)

		if s.ReadingData {
			if line == "." {
				writeLine(w, "250 OK")
				slog.Info(fmt.Sprintf("Email received %#v", s))

				// Reset session for next email
				s.DataBuffer = nil
				s.MailFrom = ""
				s.RcptTo = nil
				s.ReadingData = false
			} else {
				s.DataBuffer = append(s.DataBuffer, line)
			}

			continue
		}

		switch {
		case strings.HasPrefix(upper, "EHLO"):
			s.HeloReceived = true
			writeLine(w, fmt.Sprintf("250-%s greets %s", hostname, line[len("EHLO "):]))

		case strings.HasPrefix(upper, "MAIL FROM:"):
			if !s.HeloReceived {
				writeLine(w, "503 Bad sequence: HELO required first")
				continue
			}
			s.MailFrom = strings.TrimPrefix(line, "MAIL FROM:")
			s.MailFrom = strings.Trim(s.MailFrom, "<> ")
			writeLine(w, "250 OK")

		case strings.HasPrefix(upper, "RCPT TO:"):
			if !s.HeloReceived {
				writeLine(w, "503 Bad sequence: MAIL FROM required first")
				continue
			}

			recipient := strings.TrimPrefix(line, "RCPT TO:")
			recipient = strings.Trim(recipient, "<> ")
			//TODO check if recipient exists ("550 No such user here"
			s.RcptTo = append(s.RcptTo, recipient)
			writeLine(w, "250 OK")

		case upper == "DATA":
			if len(s.RcptTo) == 0 {
				writeLine(w, "503 Bad sequence: RCPT TO required before DATA")
				continue
			}
			s.ReadingData = true
			writeLine(w, "354 Start mail input; end with <CRLF>.<CRLF>")

		case upper == "RSET":
			s = &Session{} // Reset session
			writeLine(w, "250 OK")

		case upper == "QUIT":
			writeLine(w, fmt.Sprintf("221 %s Service closing transmission channel", hostname))
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
