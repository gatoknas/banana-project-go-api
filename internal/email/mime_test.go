package email

import (
	"strings"
	"testing"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name        string
		rawMessage  string
		wantSender  string
		wantSubject string
		wantBody    string
		wantErr     bool
	}{
		{
			name: "plain text message",
			rawMessage: "From: sender@example.com\r\n" +
				"Subject: Test Subject\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Hello World",
			wantSender:  "sender@example.com",
			wantSubject: "Test Subject",
			wantBody:    "Hello World",
			wantErr:     false,
		},
		{
			name: "html text message",
			rawMessage: "From: html@example.com\r\n" +
				"Subject: HTML Subject\r\n" +
				"Content-Type: text/html\r\n" +
				"\r\n" +
				"<h1>Hello HTML</h1>",
			wantSender:  "html@example.com",
			wantSubject: "HTML Subject",
			wantBody:    "<h1>Hello HTML</h1>",
			wantErr:     false,
		},
		{
			name: "multipart alternative message",
			rawMessage: "From: multi@example.com\r\n" +
				"Subject: Multipart Test\r\n" +
				"Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n" +
				"\r\n" +
				"--boundary123\r\n" +
				"Content-Type: text/plain\r\n" +
				"\r\n" +
				"Plain text version\r\n" +
				"--boundary123\r\n" +
				"Content-Type: text/html\r\n" +
				"\r\n" +
				"<b>HTML version</b>\r\n" +
				"--boundary123--\r\n",
			wantSender:  "multi@example.com",
			wantSubject: "Multipart Test",
			wantBody:    "<b>HTML version</b>",
			wantErr:     false,
		},
		{
			name:       "invalid message format",
			rawMessage: "Invalid-Malformed-Raw",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage([]byte(tt.rawMessage))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && msg != nil {
				if msg.Sender != tt.wantSender {
					t.Errorf("Sender = %q, want %q", msg.Sender, tt.wantSender)
				}
				if msg.Subject != tt.wantSubject {
					t.Errorf("Subject = %q, want %q", msg.Subject, tt.wantSubject)
				}
				if !strings.Contains(msg.Body, tt.wantBody) {
					t.Errorf("Body = %q, want containing %q", msg.Body, tt.wantBody)
				}
			}
		})
	}
}
