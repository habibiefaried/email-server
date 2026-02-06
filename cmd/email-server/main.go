package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	var pgStore *storage.PostgresStorage
	dbURL := os.Getenv("DB_URL")
	if dbURL != "" {
		var err error
		pgStore, err = storage.NewPostgresStorage(dbURL)
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

	// Get SMTP port from environment variable, default to 2525
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "2525"
	}

	// Always run the email server
	go server.RunSMTPServer(fqdn, smtpPort, store)

	// HTTP API setup
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "48080"
	}
	addr := ":" + port

	// Health check endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "OK")
	})

	// Inbox API endpoint
	http.HandleFunc("/inbox", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Set CORS headers immediately
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		// Check if postgres is available
		if pgStore == nil {
			http.Error(w, "Postgres storage not configured", http.StatusServiceUnavailable)
			return
		}

		// Get email address from query parameter
		address := r.URL.Query().Get("name")
		if address == "" {
			http.Error(w, "Missing 'name' query parameter", http.StatusBadRequest)
			return
		}

		// Get pagination parameters with defaults
		limit := 100
		offset := 0

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
				limit = parsedLimit
			}
		}

		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Fetch emails from postgres
		emails, err := pgStore.GetEmailsByAddress(address, limit, offset)
		if err != nil {
			log.Printf("Error fetching emails for %s: %v", address, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Return JSON response
		if err := json.NewEncoder(w).Encode(emails); err != nil {
			log.Printf("Error encoding JSON: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Handle CORS preflight requests
	http.HandleFunc("/inbox/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})

	log.Printf("Starting HTTP API on %s", addr)
	log.Printf("Endpoints: / (health check), /inbox?name=<address> (fetch emails)")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
