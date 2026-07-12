package tron

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

// testSeedHex is the BIP39 seed of the standard test mnemonic
// ("abandon" x11 + "about", empty passphrase) — a published constant.
const testSeedHex = "5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc19a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4"

// testAccountKey derives the BIP44 TRON account m/44'/195'/0' from the test
// seed, returning (xprv, xpub) strings.
func testAccountKey(t *testing.T) (string, string) {
	t.Helper()
	seed, err := hex.DecodeString(testSeedHex)
	if err != nil {
		t.Fatalf("decode seed: %v", err)
	}
	master, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	key := master
	for _, step := range []uint32{44, 195, 0} {
		if key, err = key.Derive(hdkeychain.HardenedKeyStart + step); err != nil {
			t.Fatalf("derive %d': %v", step, err)
		}
	}
	xpub, err := key.Neuter()
	if err != nil {
		t.Fatalf("neuter: %v", err)
	}
	return key.String(), xpub.String()
}

// TestDeriveAddressVectors checks derivation against the published TRON
// addresses for the standard test mnemonic (verifiable in any BIP39/44 tool,
// e.g. TronLink or iancoleman.io/bip39 with coin TRX).
func TestDeriveAddressVectors(t *testing.T) {
	_, xpub := testAccountKey(t)

	vectors := []struct {
		index uint32
		want  string
	}{
		// Published m/44'/195'/0'/0/0 vector for the test mnemonic — verified
		// against external tooling; proves the whole derivation pipeline.
		{0, "TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH"},
		// Regression pin (same code path, next index).
		{1, "TSeJkUh4Qv67VNFwY8LaAxERygNdy6NQZK"},
	}
	for _, v := range vectors {
		got, err := DeriveAddress(xpub, v.index)
		if err != nil {
			t.Fatalf("DeriveAddress(%d): %v", v.index, err)
		}
		if got != v.want {
			t.Errorf("index %d: got %s, want %s", v.index, got, v.want)
		}
	}
}

func TestDeriveAddressRejectsPrivateKey(t *testing.T) {
	xprv, _ := testAccountKey(t)
	if _, err := DeriveAddress(xprv, 0); err == nil {
		t.Fatal("expected an error for an xprv input, got nil")
	}
}

// TestDeriveAddressExternalChainKey: a depth-4 export (the external chain
// m/44'/195'/0'/0 — iancoleman's "BIP32 Extended Public Key") must derive the
// exact same addresses as the depth-3 account export.
func TestDeriveAddressExternalChainKey(t *testing.T) {
	xprv, xpub := testAccountKey(t)

	// Build the external-chain xpub (one non-hardened step below the account).
	acct, err := hdkeychain.NewKeyFromString(xprv)
	if err != nil {
		t.Fatal(err)
	}
	extPriv, err := acct.Derive(0)
	if err != nil {
		t.Fatal(err)
	}
	extPub, err := extPriv.Neuter()
	if err != nil {
		t.Fatal(err)
	}
	if extPub.Depth() != 4 {
		t.Fatalf("external-chain key depth = %d, want 4", extPub.Depth())
	}

	for _, index := range []uint32{0, 1, 7} {
		fromAccount, err := DeriveAddress(xpub, index)
		if err != nil {
			t.Fatalf("account-key derive %d: %v", index, err)
		}
		fromExternal, err := DeriveAddress(extPub.String(), index)
		if err != nil {
			t.Fatalf("external-key derive %d: %v", index, err)
		}
		if fromAccount != fromExternal {
			t.Errorf("index %d: account-key %s != external-key %s", index, fromAccount, fromExternal)
		}
	}
}

// TestDeriveAddressRejectsWrongDepth: keys above the account or below the
// external chain would derive addresses no wallet displays — refuse them.
func TestDeriveAddressRejectsWrongDepth(t *testing.T) {
	xprv, _ := testAccountKey(t)
	acct, err := hdkeychain.NewKeyFromString(xprv)
	if err != nil {
		t.Fatal(err)
	}

	// Depth 5: an address-level key (m/44'/195'/0'/0/0).
	ext, err := acct.Derive(0)
	if err != nil {
		t.Fatal(err)
	}
	addr0, err := ext.Derive(0)
	if err != nil {
		t.Fatal(err)
	}
	addrPub, err := addr0.Neuter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeriveAddress(addrPub.String(), 0); err == nil {
		t.Error("depth-5 (address-level) key accepted, want refusal")
	}

	// Depth 0: a master key.
	seed, err := hex.DecodeString(testSeedHex)
	if err != nil {
		t.Fatal(err)
	}
	master, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	masterPub, err := master.Neuter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeriveAddress(masterPub.String(), 0); err == nil {
		t.Error("depth-0 (master) key accepted, want refusal")
	}
}

func TestDeriveAddressRejectsGarbage(t *testing.T) {
	if _, err := DeriveAddress("not-an-xpub", 0); err == nil {
		t.Fatal("expected an error for a malformed key, got nil")
	}
}

func TestValidateAddress(t *testing.T) {
	valid := []string{
		"TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH", // derived vector above
		"TLRaHegyg2grMQqX85nJyCzbdRtvM5nCDn", // real-world shared deposit address
	}
	for _, addr := range valid {
		if err := ValidateAddress(addr); err != nil {
			t.Errorf("ValidateAddress(%s): unexpected error %v", addr, err)
		}
	}

	invalid := map[string]string{
		"bad checksum":     "TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdX",
		"eth-style hex":    "0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc",
		"bitcoin prefix":   "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", // version 0x00, not 0x41
		"empty":            "",
		"not base58 chars": "T!!!invalid!!!",
	}
	for name, addr := range invalid {
		if err := ValidateAddress(addr); err == nil {
			t.Errorf("%s: ValidateAddress(%q) = nil, want error", name, addr)
		}
	}
}
