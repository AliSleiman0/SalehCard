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

func TestDeriveAddressRejectsGarbage(t *testing.T) {
	if _, err := DeriveAddress("not-an-xpub", 0); err == nil {
		t.Fatal("expected an error for a malformed key, got nil")
	}
}
