package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/emersion/go-smtp"
)

type Backend struct{}

func (bkd *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

type Session struct{}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	log.Printf("Mail from: %s\n", from)
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	log.Printf("Rcpt to: %s\n", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	log.Printf("Email body: %s\n", string(body))
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

func main() {
	be := &Backend{}
	s := smtp.NewServer(be)

	// Get FQDN from environment variable, default to localhost
	fqdn := os.Getenv("FQDN")
	if fqdn == "" {
		fqdn = "localhost"
	}

	s.Addr = ":25"
	s.Domain = fqdn
	s.AllowInsecureAuth = true

	log.Printf("Starting SMTP server on %s\n", s.Addr)
	log.Printf("FQDN: %s\n", fqdn)

	// Print required DNS records
	printDNSRecords(fqdn)

	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start SMTP server on port 25: %v\n", err)
	}
}

func printDNSRecords(fqdn string) {
	if fqdn == "localhost" {
		log.Println("\n⚠️  Using localhost - this is for testing only.")
		log.Println("To receive external emails, set the FQDN environment variable:")
		log.Println("  export FQDN=mail.yourdomain.com")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("DNS RECORDS REQUIRED FOR EMAIL DELIVERY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\nFQDN: %s\n\n", fqdn)

	fmt.Println("1. A RECORD (Point domain to IP)")
	fmt.Printf("   Name:  %s\n", fqdn)
	fmt.Println("   Type:  A")
	fmt.Println("   Value: [YOUR_PUBLIC_IP] (e.g., 1.2.3.4)")

	fmt.Println("\n2. MX RECORD (Mail exchange record)")
	fmt.Println("   Name:     yourdomain.com")
	fmt.Println("   Type:     MX")
	fmt.Printf("   Value:    %s\n", fqdn)
	fmt.Println("   Priority: 10")

	fmt.Println("\n3. PTR RECORD (Reverse DNS - contact your ISP)")
	fmt.Println("   This is configured by your ISP in their reverse DNS zone")
	fmt.Println("   Reverse Zone: [REVERSED_IP].in-addr.arpa")
	fmt.Printf("   Value:        %s\n", fqdn)
	fmt.Println("   Example: If IP is 1.2.3.4, set 4.3.2.1.in-addr.arpa PTR record")

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TESTING DNS RECORDS:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("nslookup [YOUR_PUBLIC_IP]")
	fmt.Println("dig -x [YOUR_PUBLIC_IP]")
	fmt.Println("nslookup -type=MX yourdomain.com")
	fmt.Println(strings.Repeat("=", 60) + "\n")
}
