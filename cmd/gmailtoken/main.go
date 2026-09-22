// Command gmailtoken runs a one-off local OAuth 2.0 flow to obtain a Gmail
// refresh token for the email-receipt ingestion integration.
//
// Usage:
//
//	go run ./cmd/gmailtoken
//
// It reads GMAIL_CLIENT_ID and GMAIL_CLIENT_SECRET from the environment (.env
// is loaded automatically). A browser URL is printed; after granting access the
// refresh token is printed for you to paste into .env.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

func main() {
	_ = godotenv.Load()

	clientID := flag.String("client-id", os.Getenv("GMAIL_CLIENT_ID"), "OAuth client ID")
	clientSecret := flag.String("client-secret", os.Getenv("GMAIL_CLIENT_SECRET"), "OAuth client secret")
	port := flag.String("port", "8085", "local loopback port for the OAuth redirect")
	flag.Parse()

	if *clientID == "" || *clientSecret == "" {
		log.Fatal("GMAIL_CLIENT_ID and GMAIL_CLIENT_SECRET must be set (in .env or via -client-id / -client-secret)")
	}

	redirectURL := fmt.Sprintf("http://localhost:%s/oauth2callback", *port)

	conf := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{gmail.GmailReadonlyScope},
	}

	codeCh := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing 'code' query parameter", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "Authorization received. You can close this tab and return to the terminal.")
		codeCh <- code
	})

	server := &http.Server{Addr: ":" + *port, Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("local server error: %v", err)
		}
	}()

	authURL := conf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("1. Open this URL in your browser and sign in with the inbox account:")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("2. Grant access, then return here. Waiting for the callback...")

	code := <-codeCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)

	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("failed to exchange authorization code: %v", err)
	}

	if token.RefreshToken == "" {
		log.Fatal("no refresh token returned. Revoke the app's prior access in your Google Account and run this again")
	}

	fmt.Println()
	fmt.Println("Success. Add this line to .env:")
	fmt.Println()
	fmt.Printf("GMAIL_REFRESH_TOKEN=%s\n", token.RefreshToken)
}
