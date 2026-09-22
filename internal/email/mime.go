package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"
)

// Message holds the parsed header fields and decoded body of an inbound email.
type Message struct {
	Sender  string
	Subject string
	Date    time.Time
	Body    string // decoded text/html (or text/plain fallback)
}

// ParseMessage parses a raw RFC822 message into its header fields and decoded body.
func ParseMessage(raw []byte) (*Message, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	subject := msg.Header.Get("Subject")
	if decoded, err := new(mime.WordDecoder).DecodeHeader(subject); err == nil {
		subject = decoded
	}

	// The Date header is best-effort; callers fall back to Gmail's internalDate.
	date, _ := msg.Header.Date()

	body, err := findBody(msg.Header, msg.Body)
	if err != nil {
		return nil, err
	}

	return &Message{
		Sender:  strings.TrimSpace(msg.Header.Get("From")),
		Subject: strings.TrimSpace(subject),
		Date:    date,
		Body:    body,
	}, nil
}

// findBody walks a (possibly nested multipart) MIME message and returns the
// decoded text/html part, falling back to text/plain when no HTML is present.
func findBody(header mail.Header, body io.Reader) (string, error) {
	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return "", fmt.Errorf("failed to parse Content-Type: %w", err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return "", fmt.Errorf("multipart message missing boundary")
		}

		reader := multipart.NewReader(body, boundary)
		var plainFallback string
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("failed to read multipart body: %w", err)
			}

			partType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			if strings.HasPrefix(partType, "multipart/") {
				decoded, err := decodePart(mail.Header(part.Header), part)
				if err != nil {
					continue
				}
				if nested, err := findBody(mail.Header(part.Header), bytes.NewReader(decoded)); err == nil && nested != "" {
					return nested, nil
				}
				continue
			}

			decoded, err := decodePart(mail.Header(part.Header), part)
			if err != nil {
				continue
			}

			switch partType {
			case "text/html":
				return string(decoded), nil
			case "text/plain":
				if plainFallback == "" {
					plainFallback = string(decoded)
				}
			}
		}

		if plainFallback != "" {
			return plainFallback, nil
		}
		return "", fmt.Errorf("no text/html or text/plain body found")
	}

	if mediaType == "text/html" || mediaType == "text/plain" {
		decoded, err := decodePart(header, body)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	return "", fmt.Errorf("unsupported content type: %s", mediaType)
}

// decodePart applies the part's Content-Transfer-Encoding to its body.
func decodePart(header mail.Header, body io.Reader) ([]byte, error) {
	encoding := strings.ToLower(strings.TrimSpace(header.Get("Content-Transfer-Encoding")))
	switch encoding {
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(body))
	case "base64":
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		trimmed := bytes.TrimSpace(data)
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(trimmed)))
		n, err := base64.StdEncoding.Decode(decoded, trimmed)
		if err != nil {
			return nil, err
		}
		return decoded[:n], nil
	default:
		return io.ReadAll(body)
	}
}
