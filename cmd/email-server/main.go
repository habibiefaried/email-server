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

	// Initialize storage backends
	fileStore := storage.NewFileStorage("emails")

	// Try to initialize postgres storage if DB_URL is provided
	var store storage.Storage
	dbURL := os.Getenv("DB_URL")
	if dbURL != "" {
		pgStore, err := storage.NewPostgresStorage(dbURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to postgres: %v", err)
			log.Printf("Falling back to file-only storage")
			store = fileStore
		} else {
			log.Printf("Postgres storage initialized, using composite storage")
			store = storage.NewCompositeStorage(fileStore, pgStore)
		}
	} else {
		log.Printf("DB_URL not set, using file-only storage")
		store = fileStore
	}

	// Always run the email server
	go server.RunSMTPServer(fqdn, store)

	// Single HTTP API for status checks
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "48080"
	}
	addr := ":" + port
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "OK")
	})
	log.Printf("Starting HTTP API on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
