package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/habibiefaried/email-server/internal/dnsutil"
	"github.com/habibiefaried/email-server/internal/server"
	"github.com/habibiefaried/email-server/internal/storage"
)

func main() {
	mailServers := os.Getenv("MAIL_SERVERS")
	var fqdn string

	if mailServers != "" {
		pairs := strings.Split(mailServers, ":")

		for _, pair := range pairs {
			fields := strings.Split(pair, ",")
			if len(fields) != 2 {
				log.Fatalf("Invalid MAIL_SERVERS entry: %s", pair)
			}
			fqdnVal := fields[0]
			ip := fields[1]
			if err := dnsutil.ValidateFQDN(fqdnVal); err != nil {
				log.Fatalf("Invalid FQDN: %v", err)
			}
			if err := dnsutil.ValidateIPv4(ip); err != nil {
				log.Fatalf("Invalid IP: %v", err)
			}
			if err := dnsutil.PrintDNSRecords(fqdnVal, ip); err != nil {
				log.Fatalf("Configuration error for %s,%s: %v", fqdnVal, ip, err)
			}
		}

		// Use the first FQDN for the SMTP server
		fields := strings.Split(pairs[0], ",")
		fqdn = fields[0]
	} else {
		log.Printf("MAIL_SERVERS not set — Email server is running without FQDN")
	}

	// Always run the email server
	store := storage.NewFileStorage("emails")
	go server.RunSMTPServer(fqdn, store)

	// Single HTTP API for status checks
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "OK")
	})
	log.Printf("Starting HTTP API on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
