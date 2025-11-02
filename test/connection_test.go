package test

import (
	"os"
	"testing"

	"github.com/emersion/go-imap/client"
)

func TestConnection(t *testing.T) {
	username := os.Getenv("EMAIL_USERNAME")
	password := os.Getenv("EMAIL_PASSWORD")
	host := os.Getenv("IMAP_HOST")

	if username == "" || password == "" {
		t.Fatal("Set EMAIL_USERNAME and EMAIL_PASSWORD environment variables")
	}

	if host == "" {
		host = "localhost"
	}

	// Connect to server
	c, err := client.DialTLS(host+":993", nil)
	if err != nil {
		t.Fatal("DialTLS:", err)
	}
	defer c.Logout()

	// Login
	if err := c.Login(username, password); err != nil {
		t.Fatal("Login:", err)
	}

	t.Log("Successfully connected and logged in!")
}
