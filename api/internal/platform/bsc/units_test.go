package bsc

import "testing"

func TestWeiToMicros(t *testing.T) {
	cases := []struct {
		value  string
		micros int64
		ok     bool
	}{
		{"1000000000000", 1, true},                        // exactly 1 micro
		{"10000001000000000000", 10_000_001, true},        // 10.000001 USDT — salt digits intact
		{"10000001000000400321", 10_000_001, true},        // sub-micro dust floors away
		{"10000000000000000000000", 10_000_000_000, true}, // 10,000 USDT = 1e22 wei > int64
		{"999999999999", 0, false},                        // below 1 micro → quotient 0
		{"0", 0, false},
		{"-1000000000000", 0, false},
		{"not-a-number", 0, false},
		// quotient itself past int64 (absurd amount) → rejected, not wrapped
		{"10000000000000000000000000000000", 0, false},
	}
	for _, c := range cases {
		got, ok := weiToMicros(c.value)
		if ok != c.ok || got != c.micros {
			t.Errorf("weiToMicros(%q) = (%d, %v), want (%d, %v)", c.value, got, ok, c.micros, c.ok)
		}
	}
}

func TestHexWeiToMicros(t *testing.T) {
	cases := []struct {
		value  string
		micros int64
		ok     bool
	}{
		// 1 micro = 1e12 wei = 0xe8d4a51000
		{"0xe8d4a51000", 1, true},
		// 10.000001 USDT = 10000001e12 wei — salt digits intact. The 32-byte
		// zero-padded shape eth_getLogs actually returns:
		{"0x0000000000000000000000000000000000000000000000008ac723ed5e8d1000", 10_000_001, true},
		// 10,000 USDT = 1e22 wei > int64
		{"0x21e19e0c9bab2400000", 10_000_000_000, true},
		// sub-micro dust floors away (1e12 + 400321 wei)
		{"0xe8d4ab2bc1", 1, true},
		{"0x0", 0, false},
		{"0x", 0, false},
		{"", 0, false},
		{"0xzz", 0, false},
		// quotient exactly 2^63 (2^63 × 1e12 wei) → past int64, rejected
		{"0x746a5288000000000000000000", 0, false},
	}
	for _, c := range cases {
		got, ok := hexWeiToMicros(c.value)
		if ok != c.ok || got != c.micros {
			t.Errorf("hexWeiToMicros(%q) = (%d, %v), want (%d, %v)", c.value, got, ok, c.micros, c.ok)
		}
	}
}
