package tron

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"golang.org/x/crypto/sha3"
)

// tronAddressPrefix is the TRON mainnet address version byte (base58 'T...').
const tronAddressPrefix = 0x41

// ValidateAddress checks that addr is a well-formed TRON mainnet base58check
// address (version byte 0x41, 20-byte payload). Used as the boot smoke check
// for a configured shared deposit address — a typo'd address would silently
// send customer funds somewhere unrecoverable, so fail fast and loudly.
func ValidateAddress(addr string) error {
	payload, version, err := base58.CheckDecode(addr)
	if err != nil {
		return fmt.Errorf("tron: invalid address %q: %w", addr, err)
	}
	if version != tronAddressPrefix {
		return fmt.Errorf("tron: address %q is not a TRON mainnet address (version 0x%02x, want 0x41)", addr, version)
	}
	if len(payload) != 20 {
		return fmt.Errorf("tron: address %q has a %d-byte payload, want 20", addr, len(payload))
	}
	return nil
}

// DeriveAddress derives the TRON base58check address for external index i from
// a watch-only xpub. Two export levels are accepted, distinguished by BIP32
// depth — real-world exports come in both shapes:
//   - depth 3: the BIP44 account m/44'/195'/0' ("Account Extended Public Key")
//     — the standard external chain 0/i is derived below it;
//   - depth 4: the external chain m/44'/195'/0'/0 ("BIP32 Extended Public Key"
//     on tooling like iancoleman) — addresses i are derived directly.
//
// Any other depth is refused: deriving 0/i under a mis-levelled key yields
// valid-looking addresses the owner's wallet never displays, silently
// stranding customer funds. The boot smoke check prints address 0 so the
// owner can confirm it matches their wallet before the feature goes live.
//
// TRON addresses are Ethereum-style: Keccak256 of the uncompressed secp256k1
// public key (without the 0x04 tag), last 20 bytes, 0x41-prefixed, base58check.
//
// Note: wallets exporting a TRON account xpub use Bitcoin xpub version bytes
// (BIP32 doesn't vary them per coin), so no network matching is enforced.
func DeriveAddress(xpub string, index uint32) (string, error) {
	key, err := hdkeychain.NewKeyFromString(xpub)
	if err != nil {
		return "", fmt.Errorf("tron: parse xpub: %w", err)
	}
	if key.IsPrivate() {
		// A pasted xprv would put spend keys on the server — refuse loudly.
		return "", errors.New("tron: key is a PRIVATE key (xprv); export the account xpub instead")
	}

	var external *hdkeychain.ExtendedKey
	switch key.Depth() {
	case 3: // account key → derive the external (receive) chain below it
		if external, err = key.Derive(0); err != nil {
			return "", fmt.Errorf("tron: derive external chain: %w", err)
		}
	case 4: // already the external-chain key
		external = key
	default:
		return "", fmt.Errorf(
			"tron: xpub depth %d is not a supported export level — provide the account key (m/44'/195'/0', depth 3) or the external-chain key (m/44'/195'/0'/0, depth 4)",
			key.Depth())
	}
	child, err := external.Derive(index)
	if err != nil {
		return "", fmt.Errorf("tron: derive index %d: %w", index, err)
	}
	pub, err := child.ECPubKey()
	if err != nil {
		return "", fmt.Errorf("tron: pubkey for index %d: %w", index, err)
	}

	raw := pub.SerializeUncompressed()[1:] // drop the 0x04 tag → 64 bytes
	hash := sha3.NewLegacyKeccak256()
	hash.Write(raw)
	digest := hash.Sum(nil)

	return base58.CheckEncode(digest[len(digest)-20:], tronAddressPrefix), nil
}
