package smtp

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	// Test with custom port
	s1 := NewServer(Configuration{Hostname: "test.example.com", Port: "2525"})
	if s1.hostname != "test.example.com" || s1.port != "2525" {
		t.Errorf("Expected hostname=test.example.com and port=2525, got hostname=%s and port=%s", s1.hostname, s1.port)
	}

	// Test with default port
	s2 := NewServer(Configuration{Hostname: "test.example.com"})
	if s2.hostname != "test.example.com" || s2.port != "25" {
		t.Errorf("Expected hostname=test.example.com and port=25, got hostname=%s and port=%s", s2.hostname, s2.port)
	}
}

func TestHandleEhlo(t *testing.T) {
	server := &Server{hostname: "test.server.com"}
	session := &Session{}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	server.handlEhlo(session, writer, "EHLO client.example.com")

	if !session.HeloReceived {
		t.Error("Expected HeloReceived to be true")
	}
	if session.Hostname != "client.example.com" {
		t.Errorf("Expected session hostname to be client.example.com, got %s", session.Hostname)
	}

	expected := "250 test.server.com greets client.example.com\r\n250-AUTH LOGIN\r\n250-AUTH PLAIN\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}
}

func TestHandleHelo(t *testing.T) {
	server := &Server{hostname: "test.server.com"}
	session := &Session{}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	server.handleHelo(session, writer, "HELO client.example.com")

	if !session.HeloReceived {
		t.Error("Expected HeloReceived to be true")
	}
	if session.Hostname != "client.example.com" {
		t.Errorf("Expected session hostname to be client.example.com, got %s", session.Hostname)
	}

	expected := "250 test.server.com greets client.example.com\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}
}

func TestHandleAuthLogin(t *testing.T) {
	server := &Server{hostname: "test.server.com"}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// Test without HELO first
	session := &Session{}
	server.handleAuthLogin(session, writer, "AUTH LOGIN")

	expected := "503 Bad sequence: 'EHLO' required first\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}

	// Test with HELO
	buf.Reset()
	session.HeloReceived = true
	server.handleAuthLogin(session, writer, "AUTH LOGIN")

	if !session.AuthLogin.RequestedUsername {
		t.Error("Expected RequestedUsername to be true")
	}

	expected = "334 VXNlcm5hbWU6\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}
}

func TestHandleAuthPlain(t *testing.T) {
	server := &Server{hostname: "test.server.com"}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// Test without HELO first
	session := &Session{}
	server.handleAuthPlain(session, writer, "AUTH PLAIN")

	expected := "503 Bad sequence: 'EHLO' required first\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}

	// Test with valid credentials
	buf.Reset()
	session.HeloReceived = true

	// Create a valid AUTH PLAIN credentials string (format: \0username\0password)
	auth := []byte("\x00user\x00pass")
	encodedAuth := base64.StdEncoding.EncodeToString(auth)

	fmt.Printf("Encoded AUTH PLAIN: %s\n", encodedAuth) // Debug output

	server.handleAuthPlain(session, writer, "AUTH PLAIN "+encodedAuth)

	if !session.AuthLogin.IsAuthenticated {
		t.Error("Expected IsAuthenticated to be true")
	}
	if session.AuthLogin.Username != "user" {
		t.Errorf("Expected username 'user', got '%s'", session.AuthLogin.Username)
	}
	if session.AuthLogin.Password != "pass" {
		t.Errorf("Expected password 'pass', got '%s'", session.AuthLogin.Password)
	}

	expected = "235 Authentication successful\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}
}

func TestHandleMailFrom(t *testing.T) {
	server := &Server{hostname: "test.server.com"}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// Test without HELO first
	session := &Session{}
	session.AuthLogin.IsAuthenticated = true
	server.handleMailFrom(session, writer, "MAIL FROM:<sender@example.com>")

	expected := "503 Bad sequence: 'EHLO' required first\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}

	// Test with HELO
	buf.Reset()
	session.HeloReceived = true
	server.handleMailFrom(session, writer, "MAIL FROM:<sender@example.com>")

	if session.Mail.From != "sender@example.com" {
		t.Errorf("Expected From to be sender@example.com, got %s", session.Mail.From)
	}

	expected = "250 OK\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}
}

func TestHandleRcptTo(t *testing.T) {
	server := &Server{hostname: "test.server.com"}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// Test without HELO first
	session := &Session{}
	server.handleRcptTo(session, writer, "RCPT TO:<recipient@example.com>")

	expected := "503 Bad sequence: 'MAIL FROM' required first\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}

	// Test with HELO
	buf.Reset()
	session.HeloReceived = true
	server.handleRcptTo(session, writer, "RCPT TO:<recipient@example.com>")

	if len(session.Mail.To) != 1 || session.Mail.To[0] != "recipient@example.com" {
		t.Errorf("Expected recipient recipient@example.com, got %v", session.Mail.To)
	}

	expected = "250 OK\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}

	// Test adding multiple recipients
	buf.Reset()
	server.handleRcptTo(session, writer, "RCPT TO:<another@example.com>")

	if len(session.Mail.To) != 2 || session.Mail.To[1] != "another@example.com" {
		t.Errorf("Expected recipients [recipient@example.com, another@example.com], got %v", session.Mail.To)
	}
}

func TestHandleData(t *testing.T) {
	server := &Server{hostname: "test.server.com"}
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	// Test without recipients
	session := &Session{}
	session.HeloReceived = true
	server.handleData(session, writer, "DATA")

	expected := "503 Bad sequence: 'RCPT TO' required first\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}

	// Test with recipients
	buf.Reset()
	session.Mail.To = append(session.Mail.To, "recipient@example.com")
	server.handleData(session, writer, "DATA")

	if !session.Mail.ReadingData {
		t.Error("Expected ReadingData to be true")
	}

	expected = "354 Start mail input; end with <CRLF>.<CRLF>\r\n"
	if buf.String() != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, buf.String())
	}
}

func TestWriteLine(t *testing.T) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	writeLine(writer, "Test message")

	expected := "Test message\r\n"
	if buf.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, buf.String())
	}
}

func TestFullEmailFlow(t *testing.T) {
	// Create a server with authentication disabled for testing
	server := NewServer(Configuration{
		Hostname: "test.server.com",
		Port:     "0", // Use port 0 to get a random available port
	})

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Exit if listener is closed
			}
			go server.handle(conn)
		}
	}()

	defer listener.Close()

	// Connect to the server
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer conn.Close()

	// Create reader and writer
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	// Helper function to send a command and validate response
	sendCommand := func(command, expectedPrefix string) string {
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("Failed to set deadline: %v", err)
		}

		// Send command
		if _, err := w.WriteString(command + "\r\n"); err != nil {
			t.Fatalf("Failed to send command: %v", err)
		}
		if err := w.Flush(); err != nil {
			t.Fatalf("Failed to flush writer: %v", err)
		}

		// Read response - handle multi-line responses
		var fullResponse strings.Builder
		for {
			response, err := r.ReadString('\n')
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			fullResponse.WriteString(response)

			// If this is a single line response or the last line of a multi-line response
			// (doesn't start with XXX-), then break
			trimmed := strings.TrimSpace(response)
			if len(trimmed) < 4 || trimmed[3] != '-' {
				break
			}
		}

		responseStr := fullResponse.String()
		trimmedResponse := strings.TrimSpace(responseStr)

		// Check if any line starts with the expected prefix
		lines := strings.Split(trimmedResponse, "\n")
		foundPrefix := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, expectedPrefix) {
				foundPrefix = true
				break
			}
		}

		if !foundPrefix {
			t.Fatalf("Expected response prefix '%s' in response:\n%s", expectedPrefix, responseStr)
		}

		return responseStr
	}

	// Read initial greeting
	greeting, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read greeting: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(greeting), "220") {
		t.Fatalf("Expected greeting to start with 220, got: %s", greeting)
	}

	// 1. Send EHLO - this will return a multi-line response
	ehloResponse := sendCommand("EHLO client.example.com", "250")
	t.Logf("EHLO response: %s", ehloResponse)

	// 2. Authenticate using AUTH PLAIN
	authString := base64.StdEncoding.EncodeToString([]byte("\x00user\x00pass"))
	authResponse := sendCommand("AUTH PLAIN "+authString, "235")
	t.Logf("AUTH response: %s", authResponse)

	// 3. Set sender with MAIL FROM
	fromResponse := sendCommand("MAIL FROM:<sender@example.com>", "250")
	t.Logf("MAIL FROM response: %s", fromResponse)

	// 4. Add recipient with RCPT TO
	rcptResponse := sendCommand("RCPT TO:<recipient@example.com>", "250")
	t.Logf("RCPT TO response: %s", rcptResponse)

	// 5. Send DATA command
	dataResponse := sendCommand("DATA", "354")
	t.Logf("DATA response: %s", dataResponse)

	// 6. Send email content
	emailContent := []string{
		"From: Sender <sender@example.com>",
		"To: Recipient <recipient@example.com>",
		"Subject: Test Email",
		"",
		"This is a test email.",
		"Hello, world!",
		".",
	}

	for _, line := range emailContent {
		if _, err := w.WriteString(line + "\r\n"); err != nil {
			t.Fatalf("Failed to send email content: %v", err)
		}
		if err := w.Flush(); err != nil {
			t.Fatalf("Failed to flush writer: %v", err)
		}
	}

	// Verify DATA completion
	dataEndResponse, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read DATA end response: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(dataEndResponse), "250") {
		t.Fatalf("Expected 250 response after DATA, got: %s", dataEndResponse)
	}
	t.Logf("DATA end response: %s", dataEndResponse)

	// 7. Quit the session
	quitResponse := sendCommand("QUIT", "221")
	t.Logf("QUIT response: %s", quitResponse)
}
