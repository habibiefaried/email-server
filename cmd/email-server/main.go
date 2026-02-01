package main

import (
	"log"
	"os"
	"strings"

	"github.com/habibiefaried/email-server/internal/dnsutil"
	"github.com/habibiefaried/email-server/internal/server"
	"github.com/habibiefaried/email-server/internal/storage"
)

func main() {
	mailServers := os.Getenv("MAIL_SERVERS")
	if mailServers == "" {
		log.Fatalf("MAIL_SERVERS env var is required, format: fqdn,ip[:fqdn2,ip2...] e.g. test.mail.com,1.2.3.4:another.mail.com,5.6.7.8")
	}
	pairs := strings.Split(mailServers, ":")
	for _, pair := range pairs {
		fields := strings.Split(pair, ",")
		if len(fields) != 2 {
			log.Fatalf("Invalid MAIL_SERVERS entry: %s", pair)
		}
		fqdn := fields[0]
		ip := fields[1]
		if err := dnsutil.ValidateFQDN(fqdn); err != nil {
			log.Fatalf("Invalid FQDN: %v", err)
		}
		if err := dnsutil.ValidateIPv4(ip); err != nil {
			log.Fatalf("Invalid IP: %v", err)
		}
		if err := dnsutil.PrintDNSRecords(fqdn, ip); err != nil {
			log.Fatalf("Configuration error for %s,%s: %v", fqdn, ip, err)
		}
	}

	// Use the first FQDN for the SMTP server
	fields := strings.Split(pairs[0], ",")
	fqdn := fields[0]
	store := storage.NewFileStorage("emails")
	server.RunSMTPServer(fqdn, store)
}
