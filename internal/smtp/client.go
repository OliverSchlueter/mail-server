package smtp

import (
	"bufio"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/OliverSchlueter/goutils/sloki"
)

var dkimPrivateKey *rsa.PrivateKey

func LoadDKIMPrivateKey(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return fmt.Errorf("invalid PEM data")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	dkimPrivateKey = key
	return nil
}

func SendMail(m Mail) (int, error) {
	emailsSent := 0

	for _, recipient := range m.To {
		host := recipient[strings.Index(recipient, "@")+1:]

		// TODO group recipients by host to avoid multiple lookups
		var mxes []*net.MX
		if host == "localhost" {
			mxes = []*net.MX{
				{
					Host: "localhost",
					Pref: 0, // Localhost has no preference
				},
			}
		} else {
			lookup, err := net.LookupMX(host)
			if err != nil {
				return emailsSent, fmt.Errorf("failed to lookup MX records for %s: %w", host, err)
			}
			mxes = lookup
		}

		if len(mxes) == 0 {
			// TODO maybe fallback to A records?
			return emailsSent, fmt.Errorf("no MX records found for %s", host)
		}

		signedLines, err := signMail(m)
		if err != nil {
			slog.Error("Failed to sign email", sloki.WrapError(err))
			continue // Skip sending this email
		}

		sent := false
		for _, mx := range mxes {
			if err := sendTo(m, recipient, mx.Host, signedLines); err != nil {
				slog.Warn("Failed to send email", slog.String("host", mx.Host), sloki.WrapError(err))
				continue // Try the next MX record
			} else {
				sent = true
				slog.Info("Email sent successfully", slog.String("to", recipient), slog.String("host", mx.Host))
				break // Email sent successfully, break out of the loop
			}
		}

		if sent {
			emailsSent += 1
		}
	}

	return emailsSent, nil
}

func sendTo(m Mail, rcpt, host string, signedLines []string) error {
	var addr string
	if host == "localhost" {
		addr = fmt.Sprintf("%s:2525", host)
	} else {
		addr = fmt.Sprintf("%s:25", host)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server %s: %w", host, err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// 1. Read server greeting
	if err := expectStatus(reader, "220"); err != nil {
		return fmt.Errorf("failed to read server greeting: %w", err)
	}

	// 2. Send EHLO
	clientName := "localhost"
	writeLineC(writer, fmt.Sprintf("EHLO %s", clientName))
	ehloLines, err := readMultilineResponse(reader, "250")
	if err != nil {
		return fmt.Errorf("EHLO command failed: %w", err)
	}

	// 3. Check for STARTTLS support
	starttls := false
	for _, line := range ehloLines {
		if strings.Contains(line, "STARTTLS") {
			starttls = true
			break
		}
	}

	// 4. Issue STARTTLS if supported
	if starttls {
		writeLineC(writer, "STARTTLS")
		if err := expectStatus(reader, "220"); err != nil {
			return fmt.Errorf("STARTTLS command failed: %w", err)
		}

		// 5. Wrap the connection in TLS
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: host,
		})
		defer tlsConn.Close()

		if err := tlsConn.Handshake(); err != nil {
			return fmt.Errorf("TLS handshake failed: %w", err)
		}

		// Replace reader/writer with the TLS versions
		conn = tlsConn
		reader = bufio.NewReader(tlsConn)
		writer = bufio.NewWriter(tlsConn)

		// 6. Send EHLO again after TLS is established
		writeLineC(writer, fmt.Sprintf("EHLO %s", clientName))
		if _, err := readMultilineResponse(reader, "250"); err != nil {
			return fmt.Errorf("EHLO after STARTTLS failed: %w", err)
		}
	}

	// 7. Continue SMTP transaction
	writeLineC(writer, fmt.Sprintf("MAIL FROM:<%s>", m.From))
	if err = expectStatus(reader, "250"); err != nil {
		return fmt.Errorf("MAIL FROM command failed: %w", err)
	}

	writeLineC(writer, fmt.Sprintf("RCPT TO:<%s>", rcpt))
	if err = expectStatus(reader, "250"); err != nil {
		return fmt.Errorf("RCPT TO command failed for %s: %w", rcpt, err)
	}

	writeLineC(writer, "DATA")
	if err = expectStatus(reader, "354"); err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	for _, line := range signedLines {
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}
		writeLineC(writer, line)
	}

	writeLineC(writer, ".")

	if err = expectStatus(reader, "250"); err != nil {
		return fmt.Errorf("email data submission failed: %w", err)
	}

	writeLineC(writer, "QUIT")
	if err = expectStatus(reader, "221"); err != nil {
		return fmt.Errorf("QUIT command failed: %w", err)
	}

	return nil
}

func expectStatus(r *bufio.Reader, code string) error {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		slog.Debug("S: " + line)

		line = strings.TrimRight(line, "\r\n")

		if strings.HasPrefix(line, code+" ") {
			return nil
		}

		if !strings.HasPrefix(line, code+"-") {
			return fmt.Errorf("expected status %s, got %s", code, line)
		}
	}
}

func readMultilineResponse(r *bufio.Reader, code string) ([]string, error) {
	var lines []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return lines, fmt.Errorf("failed to read response: %w", err)
		}

		slog.Debug("S: " + line)

		lines = append(lines, strings.TrimRight(line, "\r\n"))

		// If the line doesn't start with code + "-" then it's the last
		if strings.HasPrefix(line, code+" ") {
			break
		}
	}
	return lines, nil
}

func writeLineC(writer *bufio.Writer, line string) {
	_, err := writer.WriteString(line + "\r\n")
	if err != nil {
		slog.Error("Failed to write line", sloki.WrapError(err))
	}
	if err := writer.Flush(); err != nil {
		slog.Error("Failed to flush writer", sloki.WrapError(err))
	}

	slog.Debug("C: " + line)
}
