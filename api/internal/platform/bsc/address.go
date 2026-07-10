package bsc

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/sha3"
)

var evmAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// ValidateAddress checks that addr is a well-formed EVM (BSC) address. Used as
// the boot smoke check for a configured shared deposit address — a typo'd
// address would silently send customer funds somewhere unrecoverable, so fail
// fast and loudly. All-lowercase or all-uppercase hex carries no checksum
// information and is accepted as-is (custodial exchanges hand out lowercase);
// mixed-case must pass the EIP-55 checksum.
func ValidateAddress(addr string) error {
	if !evmAddressRe.MatchString(addr) {
		return fmt.Errorf("bsc: invalid address %q: want 0x followed by 40 hex characters", addr)
	}
	hex := addr[2:]
	lower := strings.ToLower(hex)
	if hex == lower || hex == strings.ToUpper(hex) {
		return nil
	}

	// EIP-55: a hex letter is uppercase iff the corresponding nibble of
	// keccak256(lowercase address) is >= 8.
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(lower))
	digest := hash.Sum(nil)
	for i := 0; i < len(hex); i++ {
		c := lower[i]
		if c >= '0' && c <= '9' {
			continue
		}
		nibble := digest[i/2]
		if i%2 == 0 {
			nibble >>= 4
		} else {
			nibble &= 0x0f
		}
		want := c
		if nibble >= 8 {
			want = c - ('a' - 'A')
		}
		if hex[i] != want {
			return fmt.Errorf("bsc: address %q fails the EIP-55 checksum — re-copy it from the wallet", addr)
		}
	}
	return nil
}
