package smtp

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
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
