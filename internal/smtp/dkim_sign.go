package smtp

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/OliverSchlueter/goutils/idgen"
	"github.com/emersion/go-msgauth/dkim"
)

func signMail(m Mail) ([]string, error) {
	var buf bytes.Buffer

	// ---- Required headers ----
	fmt.Fprintf(&buf, "From: %s\r\n", m.From)
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(m.To, ", "))
	fmt.Fprintf(&buf, "Subject: %s\r\n", m.Subject)
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Message-ID: <%s@%s>\r\n", idgen.GenerateID(20), m.Domain)
	buf.WriteString("\r\n")

	// ---- Body ----
	for _, line := range m.DataBuffer {
		buf.WriteString(line + "\r\n")
	}

	raw := buf.Bytes()

	opts := &dkim.SignOptions{
		Domain:   m.Domain, // MUST match From domain
		Selector: "mail",   // DNS selector
		Signer:   dkimPrivateKey,
		HeaderKeys: []string{
			"from",
			"to",
			"subject",
			"date",
			"message-id",
		},
	}

	var signed bytes.Buffer
	if err := dkim.Sign(&signed, bytes.NewReader(raw), opts); err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimRight(signed.String(), "\r\n"), "\r\n"), nil
}
