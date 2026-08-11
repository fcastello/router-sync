package router

import (
	"net"
	"testing"
)

func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("ParseCIDR(%q): %v", cidr, err)
	}
	return n
}

func TestCalculatePriority(t *testing.T) {
	cases := []struct {
		cidr string
		want int
	}{
		{"192.168.1.50/32", 2000},
		{"192.168.1.0/31", 2001},
		{"192.168.1.0/25", 2007},
		{"192.168.1.0/24", 2008},
		{"10.0.0.0/8", 2024},
		{"0.0.0.0/0", 2032},
	}
	for _, tc := range cases {
		got := calculatePriority(mustCIDR(t, tc.cidr))
		if got != tc.want {
			t.Errorf("calculatePriority(%s) = %d, want %d", tc.cidr, got, tc.want)
		}
	}
}

func TestCalculatePriorityHostBeatsSubnet(t *testing.T) {
	host := calculatePriority(mustCIDR(t, "192.168.1.50/32"))
	slash25 := calculatePriority(mustCIDR(t, "192.168.1.0/25"))
	slash24 := calculatePriority(mustCIDR(t, "192.168.1.0/24"))
	if !(host < slash25 && slash25 < slash24) {
		t.Fatalf("expected host(%d) < /25(%d) < /24(%d)", host, slash25, slash24)
	}
}

func TestParsePolicySource(t *testing.T) {
	host, err := parsePolicySource("192.168.1.50")
	if err != nil {
		t.Fatal(err)
	}
	if host.String() != "192.168.1.50/32" {
		t.Fatalf("host = %s, want 192.168.1.50/32", host.String())
	}
	if calculatePriority(host) != 2000 {
		t.Fatalf("host priority = %d, want 2000", calculatePriority(host))
	}

	subnet, err := parsePolicySource("192.168.1.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if subnet.String() != "192.168.1.0/24" {
		t.Fatalf("subnet = %s", subnet.String())
	}
}

func TestLineMatchesSource(t *testing.T) {
	host := mustCIDR(t, "192.168.1.50/32")
	subnet := mustCIDR(t, "192.168.1.0/24")

	if !lineMatchesSource("2000: from 192.168.1.50 lookup 99", host) {
		t.Fatal("expected bare host match")
	}
	if !lineMatchesSource("2000: from 192.168.1.50/32 lookup 99", host) {
		t.Fatal("expected /32 host match")
	}
	if lineMatchesSource("2008: from 192.168.1.0/24 lookup 99", host) {
		t.Fatal("host must not match /24 rule")
	}
	if !lineMatchesSource("2008: from 192.168.1.0/24 lookup 99", subnet) {
		t.Fatal("expected /24 match")
	}
	if lineMatchesSource("2000: from 192.168.1.0 lookup 99", subnet) {
		t.Fatal("/24 must not match bare .0 host rule")
	}
}

func TestNormalizeFromToken(t *testing.T) {
	if got := normalizeFromToken("192.168.1.50"); got != "192.168.1.50/32" {
		t.Fatalf("got %s", got)
	}
	if got := normalizeFromToken("192.168.1.0/24"); got != "192.168.1.0/24" {
		t.Fatalf("got %s", got)
	}
}
