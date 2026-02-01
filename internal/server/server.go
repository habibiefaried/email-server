package server

import (
	"log"

	"github.com/emersion/go-smtp"
)

func RunSMTPServer(fqdn string, be *Backend) {
	s := smtp.NewServer(be)
	s.Addr = ":25"
	s.Domain = fqdn
	s.AllowInsecureAuth = true

	log.Printf("Starting SMTP server on %s\n", s.Addr)
	log.Printf("FQDN: %s\n", fqdn)

	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start SMTP server: %v", err)
	}
}
