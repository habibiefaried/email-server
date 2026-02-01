package dnsutil

import (
	"fmt"
	"strings"
)

func PrintDNSRecords(fqdn, publicIP string) error {
	// Validate FQDN and IP (should already be done in main)
	// Check DNS records
	aStatus := CheckARecord(fqdn, publicIP)
	mxStatus := CheckMXRecord(fqdn, fqdn)
	ptrStatus := CheckPTRRecord(publicIP, fqdn)

	if aStatus != "✓ OK" || mxStatus != "✓ OK" {
		fmt.Println("\n" + strings.Repeat("=", 100))
		fmt.Println("DNS RECORDS VERIFICATION FAILED")
		fmt.Println(strings.Repeat("=", 100))
		fmt.Println()
		fmt.Println("TYPE   | NAME                      | VALUE                                      | STATUS")
		fmt.Println(strings.Repeat("-", 100))
		fmt.Printf("A      | %-25s | %-40s | %s\n", fqdn, publicIP, aStatus)
		fmt.Printf("MX     | %-25s | %-40s | %s\n", fqdn, fqdn, mxStatus)
		fmt.Printf("PTR    | %-25s | %-40s | %s (optional)\n", ReverseIP(publicIP), fqdn, ptrStatus)
		fmt.Println(strings.Repeat("=", 100))
		return fmt.Errorf("DNS records not properly configured")
	}

	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("DNS RECORDS VERIFICATION")
	fmt.Println(strings.Repeat("=", 100))
	fmt.Println()
	fmt.Println("TYPE   | NAME                      | VALUE                                      | STATUS")
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("A      | %-25s | %-40s | %s\n", fqdn, publicIP, aStatus)
	fmt.Printf("MX     | %-25s | %-40s | %s\n", fqdn, fqdn, mxStatus)
	if ptrStatus == "✓ OK" {
		fmt.Printf("PTR    | %-25s | %-40s | %s\n", ReverseIP(publicIP), fqdn, ptrStatus)
	} else {
		fmt.Printf("PTR    | %-25s | %-40s | %s (warning: optional but recommended)\n", ReverseIP(publicIP), fqdn, ptrStatus)
	}
	fmt.Println(strings.Repeat("=", 100) + "\n")
	return nil
}
