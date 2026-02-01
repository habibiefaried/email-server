package main

import (
	"fmt"
	"io"
	"log"
	"net"
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

	// Get PUBLIC_IP from environment variable
	publicIP := os.Getenv("PUBLIC_IP")
	if publicIP == "" {
		publicIP = "localhost"
	}

	s.Addr = ":25"
	s.Domain = fqdn
	s.AllowInsecureAuth = true

	log.Printf("Starting SMTP server on %s\n", s.Addr)
	log.Printf("FQDN: %s\n", fqdn)
	log.Printf("PUBLIC_IP: %s (Email server IP address)\n", publicIP)

	// Print required DNS records
	if err := printDNSRecords(fqdn, publicIP); err != nil {
		log.Fatalf("Configuration error: %v\n", err)
	}

	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start SMTP server on port 25: %v\n", err)
	}
}

func printDNSRecords(fqdn, publicIP string) error {
	if fqdn == "localhost" {
		log.Println("\n⚠️  Using localhost - this is for testing only.")
		log.Println("To receive external emails, set the FQDN environment variable:")
		log.Println("  export FQDN=mail.yourdomain.com")
		return nil
	}

	// Validate FQDN
	if err := isValidFQDN(fqdn); err != nil {
		return fmt.Errorf("invalid FQDN: %v", err)
	}

	// Check DNS records
	aStatus := checkARecord(fqdn, publicIP)
	mxStatus := checkMXRecord(fqdn, fqdn)
	ptrStatus := checkPTRRecord(publicIP, fqdn)

	// If A or MX check failed, return error (PTR is optional)
	if aStatus != "✓ OK" || mxStatus != "✓ OK" {
		fmt.Println("\n" + strings.Repeat("=", 100))
		fmt.Println("DNS RECORDS VERIFICATION FAILED")
		fmt.Println(strings.Repeat("=", 100))
		fmt.Println()
		fmt.Println("TYPE   | NAME                      | VALUE                                      | STATUS")
		fmt.Println(strings.Repeat("-", 100))
		fmt.Printf("A      | %-25s | %-40s | %s\n", fqdn, publicIP, aStatus)
		fmt.Printf("MX     | %-25s | %-40s | %s\n", fqdn, fqdn, mxStatus)
		fmt.Printf("PTR    | %-25s | %-40s | %s (optional)\n", strings.TrimSuffix(reversIPValue(publicIP), ".in-addr.arpa")+".in-addr.arpa", fqdn, ptrStatus)
		fmt.Println(strings.Repeat("=", 100))
		return fmt.Errorf("DNS records not properly configured")
	}

	// A and MX passed, show all including PTR status
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("DNS RECORDS VERIFICATION")
	fmt.Println(strings.Repeat("=", 100))
	fmt.Println()
	fmt.Println("TYPE   | NAME                      | VALUE                                      | STATUS")
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("A      | %-25s | %-40s | %s\n", fqdn, publicIP, aStatus)
	fmt.Printf("MX     | %-25s | %-40s | %s\n", fqdn, fqdn, mxStatus)
	if ptrStatus == "✓ OK" {
		fmt.Printf("PTR    | %-25s | %-40s | %s\n", reversIPValue(publicIP), fqdn, ptrStatus)
	} else {
		fmt.Printf("PTR    | %-25s | %-40s | %s (warning: optional but recommended)\n", reversIPValue(publicIP), fqdn, ptrStatus)
	}
	fmt.Println(strings.Repeat("=", 100) + "\n")
	return nil
}

func reversIPValue(ip string) string {
	if err := isValidIP(ip); err != nil {
		log.Fatalf("Invalid IP address: %v\n", err)
	}
	octets := strings.Split(ip, ".")
	if len(octets) == 4 {
		for i, j := 0, len(octets)-1; i < j; i, j = i+1, j-1 {
			octets[i], octets[j] = octets[j], octets[i]
		}
		return strings.Join(octets, ".") + ".in-addr.arpa"
	}
	return ip
}

func isValidIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("not a valid IPv4 or IPv6 address: %s", ip)
	}
	return nil
}

func isValidFQDN(fqdn string) error {
	if fqdn == "" {
		return fmt.Errorf("FQDN cannot be empty")
	}
	if len(fqdn) > 253 {
		return fmt.Errorf("FQDN too long (max 253 characters)")
	}
	parts := strings.Split(fqdn, ".")
	if len(parts) < 2 {
		return fmt.Errorf("FQDN must have at least 2 parts (e.g., mail.example.com)")
	}
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("FQDN has empty labels")
		}
		if len(part) > 63 {
			return fmt.Errorf("FQDN label too long: %s (max 63 characters)", part)
		}
		// Check valid characters (alphanumeric and hyphens, no leading/trailing hyphens)
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return fmt.Errorf("FQDN labels cannot start or end with hyphen: %s", part)
		}
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
				return fmt.Errorf("FQDN contains invalid character: %c", r)
			}
		}
	}
	return nil
}

func checkARecord(fqdn, expectedIP string) string {
	ips, err := net.LookupHost(fqdn)
	if err != nil {
		return fmt.Sprintf("✗ FAILED (not found)")
	}
	for _, ip := range ips {
		if ip == expectedIP {
			return "✓ OK"
		}
	}
	return fmt.Sprintf("✗ FAILED (points to %s, expected %s)", strings.Join(ips, ", "), expectedIP)
}

func checkMXRecord(domain, expectedFQDN string) string {
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return "✗ FAILED (not found)"
	}
	for _, mx := range mxRecords {
		// Remove trailing dot from MX host
		mxHost := strings.TrimSuffix(mx.Host, ".")
		if mxHost == expectedFQDN {
			return "✓ OK"
		}
	}
	var hosts []string
	for _, mx := range mxRecords {
		hosts = append(hosts, strings.TrimSuffix(mx.Host, "."))
	}
	return fmt.Sprintf("✗ FAILED (points to %s, expected %s)", strings.Join(hosts, ", "), expectedFQDN)
}

func checkPTRRecord(publicIP, expectedFQDN string) string {
	names, err := net.LookupAddr(publicIP)
	if err != nil {
		return "✗ FAILED (not found)"
	}
	for _, name := range names {
		// Remove trailing dot from PTR record
		name = strings.TrimSuffix(name, ".")
		if name == expectedFQDN {
			return "✓ OK"
		}
	}
	// Remove trailing dots for display
	var cleanedNames []string
	for _, name := range names {
		cleanedNames = append(cleanedNames, strings.TrimSuffix(name, "."))
	}
	return fmt.Sprintf("✗ FAILED (points to %s, expected %s)", strings.Join(cleanedNames, ", "), expectedFQDN)
}
