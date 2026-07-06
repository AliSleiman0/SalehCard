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

// DeriveAddress derives the TRON base58check address for external index i from
// an account-level xpub. The owner exports the BIP44 account m/44'/195'/0' as
// an xpub (watch-only — no private key material); we derive the standard
// external chain 0/i below it.
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

	external, err := key.Derive(0) // external (receive) chain
	if err != nil {
		return "", fmt.Errorf("tron: derive external chain: %w", err)
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
