package email

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Config holds the settings required to read the receipt inbox via the Gmail API.
type Config struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	TargetEmail  string
	Label        string
	SenderFilter string
}

// RawMessage is a fetched Gmail message with its decoded RFC822 payload.
type RawMessage struct {
	ID         string
	Raw        []byte
	InternalTS int64
}

// Client is a thin Gmail API client scoped to a single inbox.
type Client struct {
	svc *gmail.Service
	cfg Config
}

// NewClient builds an authenticated Gmail client from an OAuth2 refresh token.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RefreshToken == "" {
		return nil, fmt.Errorf("gmail OAuth credentials are not configured")
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailReadonlyScope},
	}

	// A token with only a refresh token will be transparently refreshed by the client.
	token := &oauth2.Token{RefreshToken: cfg.RefreshToken}
	httpClient := oauthConfig.Client(ctx, token)

	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	return &Client{svc: svc, cfg: cfg}, nil
}

func (c *Client) user() string {
	if c.cfg.TargetEmail != "" {
		return c.cfg.TargetEmail
	}
	return "me"
}

// buildQuery builds a Gmail search query for the given half-open time range.
func (c *Client) buildQuery(from, to time.Time) string {
	parts := []string{
		fmt.Sprintf("after:%d", from.Unix()),
		fmt.Sprintf("before:%d", to.Unix()),
	}
	if c.cfg.SenderFilter != "" {
		parts = append(parts, "from:"+c.cfg.SenderFilter)
	}
	if c.cfg.Label != "" {
		parts = append(parts, "label:"+c.cfg.Label)
	}
	return strings.Join(parts, " ")
}

// decodeRawMessage decodes the Gmail message.raw field. Gmail returns the
// message base64url-encoded, but padding (and occasionally the standard
// alphabet) may be present, so we normalize before decoding.
func decodeRawMessage(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "=")
	s = strings.NewReplacer("-", "+", "_", "/").Replace(s)
	return base64.RawStdEncoding.DecodeString(s)
}

// FetchMessages returns the raw RFC822 messages received in [from, to).
func (c *Client) FetchMessages(ctx context.Context, from, to time.Time) ([]RawMessage, error) {
	query := c.buildQuery(from, to)
	var messages []RawMessage

	err := c.svc.Users.Messages.List(c.user()).Q(query).Pages(ctx, func(resp *gmail.ListMessagesResponse) error {
		for _, stub := range resp.Messages {
			full, err := c.svc.Users.Messages.Get(c.user(), stub.Id).Format("raw").Do()
			if err != nil {
				return fmt.Errorf("failed to get message %s: %w", stub.Id, err)
			}
			raw, err := decodeRawMessage(full.Raw)
			if err != nil {
				return fmt.Errorf("failed to decode raw message %s: %w", stub.Id, err)
			}
			messages = append(messages, RawMessage{
				ID:         full.Id,
				Raw:        raw,
				InternalTS: full.InternalDate,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	return messages, nil
}
