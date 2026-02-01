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

	// Extract domain from FQDN (e.g., test.mail.habibiefaried.com -> habibiefaried.com)
	parts := strings.Split(fqdn, ".")
	var domain string
	if len(parts) >= 2 {
		domain = strings.Join(parts[len(parts)-2:], ".")
	} else {
		domain = fqdn
	}

	// Generate PTR record (reverse IP)
	ptrRecord, err := reversIP(publicIP)
	if err != nil {
		return fmt.Errorf("invalid PUBLIC_IP: %v", err)
	}

	// Check DNS records
	aStatus := checkARecord(fqdn, publicIP)
	mxStatus := checkMXRecord(domain, fqdn)
	ptrStatus := checkPTRRecord(publicIP, fqdn)

	// If any check failed, return error
	if aStatus != "✓ OK" || mxStatus != "✓ OK" || ptrStatus != "✓ OK" {
		fmt.Println("\n" + strings.Repeat("=", 100))
		fmt.Println("DNS RECORDS VERIFICATION FAILED")
		fmt.Println(strings.Repeat("=", 100))
		fmt.Println()
		fmt.Println("TYPE   | NAME                      | VALUE                                      | STATUS")
		fmt.Println(strings.Repeat("-", 100))
		fmt.Printf("A      | %-25s | %-40s | %s\n", fqdn, publicIP, aStatus)
		fmt.Printf("MX     | %-25s | %-40s | %s\n", domain, fqdn+" (priority: 10)", mxStatus)
		fmt.Printf("PTR    | %-25s | %-40s | %s\n", ptrRecord, fqdn, ptrStatus)
		fmt.Println(strings.Repeat("=", 100))
		return fmt.Errorf("DNS records not properly configured")
	}

	// All checks passed, print table with status
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("DNS RECORDS VERIFICATION PASSED")
	fmt.Println(strings.Repeat("=", 100))
	fmt.Println()
	fmt.Println("TYPE   | NAME                      | VALUE                                      | STATUS")
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("A      | %-25s | %-40s | %s\n", fqdn, publicIP, aStatus)
	fmt.Printf("MX     | %-25s | %-40s | %s\n", domain, fqdn+" (priority: 10)", mxStatus)
	fmt.Printf("PTR    | %-25s | %-40s | %s\n", ptrRecord, fqdn, ptrStatus)
	fmt.Println(strings.Repeat("=", 100) + "\n")
	return nil
}

func reversIP(ip string) (string, error) {
	octets := strings.Split(ip, ".")
	if len(octets) != 4 {
		return "", fmt.Errorf("not a valid IPv4 address: %s", ip)
	}
	// Verify each octet is a valid number
	for _, octet := range octets {
		if octet == "" {
			return "", fmt.Errorf("not a valid IPv4 address: %s", ip)
		}
	}
	// Reverse the octets
	for i, j := 0, len(octets)-1; i < j; i, j = i+1, j-1 {
		octets[i], octets[j] = octets[j], octets[i]
	}
	return strings.Join(octets, ".") + ".in-addr.arpa", nil
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
