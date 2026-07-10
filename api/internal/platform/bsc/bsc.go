// Package bsc is the BEP20 (BNB Smart Chain) chain-reader adapter set for
// shared-address USDT deposits. It deliberately reuses the tron package's port
// types ([tron.TransferLister], [tron.Payment], [tron.AmountHint]) — the
// payment watcher programs against that port, and BEP20 is shared-address-mode
// only, so no BSC equivalent of derived addresses or FindPayment exists.
// Adding a chain data source is a new adapter file plus one case in [New].
package bsc

import (
	"fmt"

	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
)

// USDTContractBSC is the BSC mainnet USDT (Binance-Peg Tether) BEP20 contract.
// NOTE: unlike TRC20's 6 decimals, this token has 18 — the adapter scales
// values down to micro-USDT so the payment module never changes units.
const USDTContractBSC = "0x55d398326f99059fF775485246999027B3197955"

// Config selects and configures the active BEP20 chain-reader adapter.
type Config struct {
	// Provider is one of "etherscan" or "stub" (default "stub" — dev needs
	// zero chain config; the stub auto-pays after a short delay).
	Provider  string
	Etherscan EtherscanConfig
	Stub      tron.StubConfig
}

// New builds the shared-address transfer lister for cfg.Provider. The stub
// case reuses [tron.NewStub]: its ListTransfers fabricates one payment per
// elapsed amount hint and is address- and network-agnostic, so the same dev
// stub exercises both chains end-to-end.
func New(cfg Config) (tron.TransferLister, error) {
	switch cfg.Provider {
	case "", "stub":
		return tron.NewStub(cfg.Stub), nil
	case "etherscan":
		return newEtherscan(cfg.Etherscan)
	default:
		return nil, fmt.Errorf("unknown BEP20 chain provider %q", cfg.Provider)
	}
}
