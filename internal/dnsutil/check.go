package dnsutil

import (
	"fmt"
	"net"
	"strings"
)

func CheckARecord(fqdn, expectedIP string) (bool, string) {
	ips, err := net.LookupHost(fqdn)
	if err != nil {
		return false, fmt.Sprintf("✗ FAILED (not found)")
	}
	for _, ip := range ips {
		if ip == expectedIP {
			return true, "✓ OK"
		}
	}
	return false, fmt.Sprintf("✗ FAILED (points to %s, expected %s)", strings.Join(ips, ", "), expectedIP)
}

func CheckMXRecord(fqdn, expectedFQDN string) (bool, string) {
	mxRecords, err := net.LookupMX(fqdn)
	if err != nil {
		return false, "✗ FAILED (not found)"
	}
	for _, mx := range mxRecords {
		mxHost := strings.TrimSuffix(mx.Host, ".")
		if mxHost == expectedFQDN {
			return true, "✓ OK"
		}
	}
	var hosts []string
	for _, mx := range mxRecords {
		hosts = append(hosts, strings.TrimSuffix(mx.Host, "."))
	}
	return false, fmt.Sprintf("✗ FAILED (points to %s, expected %s)", strings.Join(hosts, ", "), expectedFQDN)
}
