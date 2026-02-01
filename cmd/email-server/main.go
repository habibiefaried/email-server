package main

import (
	"log"
	"os"

	"github.com/habibiefaried/email-server/internal/dnsutil"
	"github.com/habibiefaried/email-server/internal/server"
)

func main() {
	fqdn := os.Getenv("FQDN")
	publicIP := os.Getenv("PUBLIC_IP")

	if err := dnsutil.ValidateFQDN(fqdn); err != nil {
		log.Fatalf("Invalid FQDN: %v", err)
	}
	if err := dnsutil.ValidateIPv4(publicIP); err != nil {
		log.Fatalf("Invalid PUBLIC_IP: %v", err)
	}

	if err := dnsutil.PrintDNSRecords(fqdn, publicIP); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	be := &server.Backend{}
	server.RunSMTPServer(fqdn, be)
}
