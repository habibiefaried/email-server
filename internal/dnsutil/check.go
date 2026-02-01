package dnsutil

import (
	"fmt"
	"net"
	"strings"
)

func CheckARecord(fqdn, expectedIP string) string {
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

func CheckMXRecord(fqdn, expectedFQDN string) string {
	mxRecords, err := net.LookupMX(fqdn)
	if err != nil {
		return "✗ FAILED (not found)"
	}
	for _, mx := range mxRecords {
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

func CheckPTRRecord(publicIP, expectedFQDN string) string {
	names, err := net.LookupAddr(publicIP)
	if err != nil {
		return "✗ FAILED (not found)"
	}
	for _, name := range names {
		name = strings.TrimSuffix(name, ".")
		if name == expectedFQDN {
			return "✓ OK"
		}
	}
	var cleanedNames []string
	for _, name := range names {
		cleanedNames = append(cleanedNames, strings.TrimSuffix(name, "."))
	}
	return fmt.Sprintf("✗ FAILED (points to %s, expected %s)", strings.Join(cleanedNames, ", "), expectedFQDN)
}

func ReverseIP(ip string) string {
	octets := strings.Split(ip, ".")
	if len(octets) == 4 {
		for i, j := 0, len(octets)-1; i < j; i, j = i+1, j-1 {
			octets[i], octets[j] = octets[j], octets[i]
		}
		return strings.Join(octets, ".") + ".in-addr.arpa"
	}
	return ip
}
