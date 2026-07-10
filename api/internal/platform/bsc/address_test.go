package bsc

import "testing"

func TestValidateAddress(t *testing.T) {
	valid := []string{
		// The client's real deposit address (all-lowercase — no checksum info).
		"0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc",
		// All-uppercase hex likewise carries no checksum.
		"0x5E0A66CEDC7688AAB52C87DC02BFF97F6575D7DC",
		// EIP-55 reference vectors (mixed case, valid checksums).
		"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		"0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
		// The BSC USDT contract itself is checksummed.
		USDTContractBSC,
	}
	for _, addr := range valid {
		if err := ValidateAddress(addr); err != nil {
			t.Errorf("ValidateAddress(%q) = %v, want nil", addr, err)
		}
	}

	invalid := []string{
		"",
		"TLRaHegyg2grMQqX85nJyCzbdRtvM5nCDn",           // TRON address
		"0x5e0a66cedc7688aab52c87dc02bff97f6575d7d",    // 39 hex chars
		"0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc1",  // 41 hex chars
		"5e0a66cedc7688aab52c87dc02bff97f6575d7dc",     // missing 0x
		"0x5e0a66cedc7688aab52c87dc02bff97f6575d7dg",   // non-hex char
		"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAeD",   // checksum broken (last char case flipped)
	}
	for _, addr := range invalid {
		if err := ValidateAddress(addr); err == nil {
			t.Errorf("ValidateAddress(%q) = nil, want error", addr)
		}
	}
}
