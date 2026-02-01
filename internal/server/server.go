package server

import (
	"log"

	"github.com/emersion/go-smtp"
	"github.com/habibiefaried/email-server/internal/storage"
)

func RunSMTPServer(fqdn string, store storage.Storage) {
	be := &Backend{Store: store}
	s := smtp.NewServer(be)
	s.Addr = ":25"
	// s.Domain = "" // Accept all domains
	s.AllowInsecureAuth = true

	log.Printf("Starting SMTP server on %s\n", s.Addr)
	log.Printf("FQDN: %s\n", fqdn)

	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start SMTP server: %v", err)
	}
}
