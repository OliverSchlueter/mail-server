package main

import (
	"bufio"
	"log/slog"
	"net"
)

func main() {
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

	slog.Info("New connection established", "remote_addr", conn.RemoteAddr().String())

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	writeLine(w, "Moin")

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			slog.Warn("Failed to read from connection", "error", err)
			return
		}

		slog.Info("Client: " + line)

		writeLine(w, "Echo: "+line)
	}
}

func writeLine(w *bufio.Writer, line string) {
	w.WriteString(line + "\n")
	w.Flush()
	slog.Info("Server: " + line)
}
