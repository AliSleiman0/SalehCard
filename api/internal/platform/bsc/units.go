package bsc

import (
	"math/big"
	"strings"
)

// weiPerMicro scales 18-decimal BEP20 USDT base units down to the 6-decimal
// micro-USDT unit the payment module works in (10^12).
var weiPerMicro = big.NewInt(1_000_000_000_000)

// bigToMicros converts an 18-decimal base-unit amount into micro-USDT,
// flooring away sub-micro dust (< 10^12 wei ≈ $0.000001). Flooring — rather
// than rejecting non-multiples — means a payment carrying stray dust still
// exact-matches its intent instead of stranding in the reconciliation queue;
// the value lost is under a millionth of a dollar. big.Int is mandatory:
// 10,000 USDT is 10^22 base units, past int64.
func bigToMicros(wei *big.Int) (int64, bool) {
	if wei == nil || wei.Sign() <= 0 {
		return 0, false
	}
	micros := new(big.Int).Quo(wei, weiPerMicro)
	if !micros.IsInt64() || micros.Sign() <= 0 {
		return 0, false
	}
	return micros.Int64(), true
}

// weiToMicros parses a DECIMAL integer value string (the Etherscan tokentx
// shape) into micro-USDT.
func weiToMicros(value string) (int64, bool) {
	wei, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return 0, false
	}
	return bigToMicros(wei)
}

// hexWeiToMicros parses a 0x-HEX quantity (the eth_getLogs `data` shape) into
// micro-USDT.
func hexWeiToMicros(value string) (int64, bool) {
	v := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if v == "" {
		return 0, false
	}
	wei, ok := new(big.Int).SetString(v, 16)
	if !ok {
		return 0, false
	}
	return bigToMicros(wei)
}
