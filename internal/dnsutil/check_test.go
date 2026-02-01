package dnsutil

import (
	"net"
	"testing"
)

func TestReverseIP(t *testing.T) {
	if got := ReverseIP("1.2.3.4"); got != "4.3.2.1.in-addr.arpa" {
		t.Errorf("ReverseIP failed: got %q", got)
	}
}

// The following tests are for demonstration and will only pass if the test environment
// has the relevant DNS records. They are not suitable for CI unless you mock net.Lookup*.
func TestCheckARecord_Localhost(t *testing.T) {
	if CheckARecord("localhost", "127.0.0.1") != "✓ OK" {
		t.Error("CheckARecord failed for localhost")
	}
}

func TestCheckPTRRecord_Localhost(t *testing.T) {
	addrs, _ := net.LookupHost("localhost")
	if len(addrs) == 0 {
		t.Skip("No localhost address found")
	}
	if CheckPTRRecord(addrs[0], "localhost") == "" {
		t.Error("CheckPTRRecord failed for localhost")
	}
}
