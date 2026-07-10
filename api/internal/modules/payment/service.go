package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

const (
	defaultIntentExpiry = 30 * time.Minute
	defaultLateGrace    = 7 * 24 * time.Hour
	defaultMaxPending   = 3
	// maxIntentUSD mirrors the top-up queue's maxTopUpAmount cap.
	maxIntentUSD = 10_000
	// excessCreditMinMicros: an order over-payment below one cent is dust —
	// not worth a ledger row.
	excessCreditMinMicros = 10_000
)

// topUpCrediter is the slice of the wallet module settlement needs (satisfied
// by *wallet.WalletService.TopUp).
type topUpCrediter interface {
	TopUp(ctx context.Context, userID bson.ObjectID, amount float64, method, ref string) (*wallet.WalletTransaction, error)
}

// OrderSettler is the slice of the order module settlement needs. Implemented
// by *order.OrderService and wired post-construction (SetOrderSettler) in
// server.go — order imports payment for intent creation, so this side of the
// loop must stay an interface to avoid the import cycle. Both methods MUST be
// idempotent: settlement retries every watcher tick until it succeeds.
type OrderSettler interface {
	// FulfillPaidOrder fulfills a usdt order whose on-chain payment confirmed.
	FulfillPaidOrder(ctx context.Context, orderID bson.ObjectID, paidUSD float64, txRef string) error
	// FailUnpaidOrder marks a usdt order failed (expiry / underpayment).
	FailUnpaidOrder(ctx context.Context, orderID bson.ObjectID, reason string) error
}

// Config tunes the payment module. For TRC20, exactly one of XPub /
// SharedAddress is set when the network is on (config.Validate enforces this
// at boot); BEP20 is additive and shared-mode only.
type Config struct {
	// XPub is the watch-only account key deposit addresses derive from
	// (TRC20 derived mode: one unique address per intent).
	XPub string
	// SharedAddress is the single fixed TRC20 deposit address every TRC20
	// intent shares (shared mode: identity = exact salted amount).
	SharedAddress string
	// BEP20SharedAddress is the single fixed BEP20 (BSC) deposit address;
	// setting it enables the second network (always shared mode).
	BEP20SharedAddress string
	// IntentExpiry is how long a customer has to pay (default 30m).
	IntentExpiry time.Duration
	// LateGrace is how long after expiry the watcher keeps scanning an unpaid
	// address so a late transfer still lands as a wallet credit (default 7d).
	LateGrace time.Duration
	// MaxPending caps a user's open intents (default 3).
	MaxPending int
}

// Service orchestrates intent creation and settlement.
type Service struct {
	store  Store
	reader tron.Reader        // TRC20 chain reader (both address modes)
	bep20  tron.TransferLister // BEP20 shared-address lister; nil when the network is off
	wallet topUpCrediter
	orders OrderSettler // nil until SetOrderSettler; nil-checked in settle
	ntf    notification.Notifier
	cfg    Config
}

// NewService constructs the payment service. bep20 may be nil (BEP20 off).
func NewService(store Store, reader tron.Reader, bep20 tron.TransferLister, wlt topUpCrediter, ntf notification.Notifier, cfg Config) *Service {
	if cfg.IntentExpiry <= 0 {
		cfg.IntentExpiry = defaultIntentExpiry
	}
	if cfg.LateGrace <= 0 {
		cfg.LateGrace = defaultLateGrace
	}
	if cfg.MaxPending <= 0 {
		cfg.MaxPending = defaultMaxPending
	}
	if ntf == nil {
		ntf = notification.Nop{}
	}
	return &Service{store: store, reader: reader, bep20: bep20, wallet: wlt, ntf: ntf, cfg: cfg}
}

// SetOrderSettler wires the order-fulfillment port. Called once at boot after
// both services exist (order needs payment first for intent creation).
func (s *Service) SetOrderSettler(o OrderSettler) { s.orders = o }

// trc20Enabled reports whether the TRC20 network is configured (either mode).
func (s *Service) trc20Enabled() bool {
	return (s.cfg.XPub != "" || s.cfg.SharedAddress != "") && s.reader != nil
}

// bep20Enabled reports whether the BEP20 network is configured (shared only).
func (s *Service) bep20Enabled() bool {
	return s.cfg.BEP20SharedAddress != "" && s.bep20 != nil
}

// Enabled reports whether on-chain USDT payments are configured on any network.
func (s *Service) Enabled() bool { return s.trc20Enabled() || s.bep20Enabled() }

// Networks lists the enabled networks, TRC20 first (it doubles as the default
// for clients that predate network selection).
func (s *Service) Networks() []string {
	var out []string
	if s.trc20Enabled() {
		out = append(out, NetworkTRC20)
	}
	if s.bep20Enabled() {
		out = append(out, NetworkBEP20)
	}
	return out
}

// DefaultNetwork is what an intent created without an explicit network gets
// (old-app compatibility). Empty when the feature is off.
func (s *Service) DefaultNetwork() string {
	if n := s.Networks(); len(n) > 0 {
		return n[0]
	}
	return ""
}

// SupportsNetwork reports whether new intents may be created on network.
func (s *Service) SupportsNetwork(network string) bool {
	switch network {
	case NetworkTRC20:
		return s.trc20Enabled()
	case NetworkBEP20:
		return s.bep20Enabled()
	}
	return false
}

// listerFor returns the shared-address transfer lister for network (nil when
// that network can't list — logged and skipped by the watcher).
func (s *Service) listerFor(network string) tron.TransferLister {
	switch network {
	case NetworkTRC20:
		if l, ok := s.reader.(tron.TransferLister); ok {
			return l
		}
	case NetworkBEP20:
		return s.bep20
	}
	return nil
}

// methodForNetwork maps an intent/deposit network to its wallet-ledger method
// (each method carries its own unique (method, ref) dedup index).
func methodForNetwork(network string) string {
	if network == NetworkBEP20 {
		return "usdt_bep20"
	}
	return "usdt_trc20"
}

// sharedMode reports whether NEW TRC20 intents get the shared address (open
// intents keep the mode stamped on them regardless).
func (s *Service) sharedMode() bool { return s.cfg.SharedAddress != "" }

// ExpiryMinutes exposes the configured window for the client config endpoint.
func (s *Service) ExpiryMinutes() int { return int(s.cfg.IntentExpiry / time.Minute) }

// CreateTopUpIntent opens an on-chain top-up intent for the user. An empty
// network means the default (old-app compatibility).
func (s *Service) CreateTopUpIntent(ctx context.Context, userID bson.ObjectID, amountUSD float64, network, idemKey string) (*Intent, error) {
	return s.createIntent(ctx, &Intent{
		UserID:  userID,
		Purpose: PurposeTopUp,
		Network: network,
	}, amountUSD, idemKey)
}

// CreateOrderIntent opens the payment intent for a just-placed usdt order.
// PlaceOrder's own Idempotency-Key dedup covers retries at the order level, so
// the intent derives its key from the order id (one live intent per order).
func (s *Service) CreateOrderIntent(ctx context.Context, userID, orderID bson.ObjectID, amountUSD float64, network string) (*Intent, error) {
	oid := orderID
	return s.createIntent(ctx, &Intent{
		UserID:  userID,
		Purpose: PurposeOrder,
		OrderID: &oid,
		Network: network,
	}, amountUSD, "order:"+orderID.Hex())
}

func (s *Service) createIntent(ctx context.Context, in *Intent, amountUSD float64, idemKey string) (*Intent, error) {
	if !s.Enabled() {
		return nil, &apperrors.AppError{
			Code:    "PAYMENT_METHOD_UNAVAILABLE",
			Message: "USDT payments are not available right now",
			Err:     apperrors.ErrBadRequest,
		}
	}
	if amountUSD <= 0 {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "amount must be positive", Err: apperrors.ErrBadRequest}
	}
	if in.Network == "" {
		in.Network = s.DefaultNetwork()
	}
	if !s.SupportsNetwork(in.Network) {
		return nil, &apperrors.AppError{
			Code:    "BAD_REQUEST",
			Message: fmt.Sprintf("unsupported payment network %q", in.Network),
			Err:     apperrors.ErrBadRequest,
		}
	}
	if amountUSD > maxIntentUSD {
		return nil, &apperrors.AppError{
			Code:    "BAD_REQUEST",
			Message: fmt.Sprintf("amount exceeds the %d USD maximum", maxIntentUSD),
			Err:     apperrors.ErrBadRequest,
		}
	}

	// Idempotency short-circuit: a replayed request returns the intent it
	// already created (any status — the client resolves what to show).
	if idemKey != "" {
		if existing, err := s.store.GetByIdempotencyKey(ctx, in.UserID, idemKey); err == nil {
			return existing, nil
		} else if !errors.Is(err, apperrors.ErrNotFound) {
			return nil, err
		}
	}

	// Spam guard: top-ups are user-initiated and capped; order intents are
	// already capped by the order flow itself.
	if in.Purpose == PurposeTopUp {
		open, err := s.store.CountOpenForUser(ctx, in.UserID)
		if err != nil {
			return nil, err
		}
		if open >= int64(s.cfg.MaxPending) {
			return nil, &apperrors.AppError{
				Code:    "TOO_MANY_PENDING",
				Message: "you already have pending USDT payments — pay or wait for them to expire first",
				Err:     apperrors.ErrBadRequest,
			}
		}
	}

	now := time.Now().UTC()
	in.AmountExpectedMicros = USDToMicros(amountUSD)
	in.Status = StatusPending
	in.IdempotencyKey = idemKey
	in.CreatedAt = now
	in.ExpiresAt = now.Add(s.cfg.IntentExpiry)

	// BEP20 is shared-mode only; TRC20 shares when configured that way. The
	// per-network deposit address is decided HERE — insertShared is
	// address-agnostic.
	if in.Network == NetworkBEP20 {
		in.Address = s.cfg.BEP20SharedAddress
		return s.insertShared(ctx, in)
	}
	if s.sharedMode() {
		in.Address = s.cfg.SharedAddress
		return s.insertShared(ctx, in)
	}

	index, err := s.store.NextDerivationIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("payment: derivation index: %w", err)
	}
	// A burned index on any failure below is harmless — indexes are free and
	// never reused, which is exactly what keeps addresses unambiguous.
	address, err := tron.DeriveAddress(s.cfg.XPub, index)
	if err != nil {
		return nil, fmt.Errorf("payment: derive address: %w", err)
	}
	in.AddressMode = AddressModeDerived
	in.Address = address
	in.DerivationIndex = index

	if err := s.store.Insert(ctx, in); err != nil {
		if errors.Is(err, apperrors.ErrConflict) && idemKey != "" {
			// A concurrent request won the idempotency race; return the winner.
			if existing, ferr := s.store.GetByIdempotencyKey(ctx, in.UserID, idemKey); ferr == nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return in, nil
}

// maxSaltAttempts bounds the shared-mode resalt loop. A conflict needs two
// open intents landing on the same salted total (cross-base collision) — one
// retry virtually always clears it; three failures means something is wrong.
const maxSaltAttempts = 3

// insertShared completes intent creation in shared-address mode (the caller
// has set the per-network Address): identity is the exact salted amount, so
// the amount must be unique among OPEN shared intents on the intent's network
// (enforced by the unique partial (network, amount) sharedOpen index;
// ErrConflict here after the idempotency re-read means an amount collision →
// resalt and retry).
func (s *Service) insertShared(ctx context.Context, in *Intent) (*Intent, error) {
	base := in.AmountExpectedMicros
	in.AddressMode = AddressModeShared
	in.SharedOpen = true

	for range maxSaltAttempts {
		salt, err := s.store.NextAmountSalt(ctx)
		if err != nil {
			return nil, fmt.Errorf("payment: amount salt: %w", err)
		}
		in.AmountSaltMicros = salt
		in.AmountExpectedMicros = base + salt

		err = s.store.Insert(ctx, in)
		if err == nil {
			return in, nil
		}
		if !errors.Is(err, apperrors.ErrConflict) {
			return nil, err
		}
		// A conflict is either the idempotency race (same as derived mode —
		// return the winner) or an open-amount collision (fall through to a
		// fresh salt).
		if in.IdempotencyKey != "" {
			if existing, ferr := s.store.GetByIdempotencyKey(ctx, in.UserID, in.IdempotencyKey); ferr == nil {
				return existing, nil
			}
		}
	}
	return nil, &apperrors.AppError{
		Code:    "PAYMENT_BUSY",
		Message: "could not allocate a unique payment amount — please try again",
		Err:     apperrors.ErrConflict,
	}
}

// GetIntent returns an intent, enforcing ownership (a foreign intent is
// reported as not found).
func (s *Service) GetIntent(ctx context.Context, userID, id bson.ObjectID) (*Intent, error) {
	in, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.UserID != userID {
		return nil, apperrors.ErrNotFound
	}
	return in, nil
}

// GetActiveOrderIntent returns the live intent for an order (used by the
// order handler to attach deposit details to a placed/replayed usdt order).
func (s *Service) GetActiveOrderIntent(ctx context.Context, orderID bson.ObjectID) (*Intent, error) {
	return s.store.GetActiveByOrder(ctx, orderID)
}

// ErrOrderNotPayable is returned by OrderSettler.FulfillPaidOrder when the
// order can no longer be fulfilled (it already failed — e.g. the intent-expiry
// sweep or an underpayment beat this payment to it). The payment then settles
// as a wallet credit instead, so the customer's money is never stranded.
var ErrOrderNotPayable = errors.New("payment: order is not payable")

// settle applies a claimed (confirming) intent: move the money, mark the
// intent confirmed, notify. It is retried by the watcher until it succeeds,
// so every step is idempotent: the wallet credit dedupes on the intent-scoped
// ledger ref (unique partial index on wallet_transactions), order fulfillment
// dedupes on the order-status claim, and which branch an order payment takes
// is decided by the ORDER's own atomic transition (ErrOrderNotPayable), not
// by guessing clocks.
func (s *Service) settle(ctx context.Context, in *Intent) error {
	if in.Purpose == PurposeTopUp {
		if err := s.creditWallet(ctx, in, in.AmountReceivedMicros, in.ID.Hex(), SettlementWalletTopUp); err != nil {
			return err
		}
		s.notify(ctx, in, notification.KindPaymentConfirmed, "Wallet topped up",
			fmt.Sprintf("Your USDT payment of $%.2f was confirmed and added to your wallet.", MicrosToUSD(in.AmountReceivedMicros)))
		return nil
	}

	// Order purpose from here on.
	if s.orders == nil || in.OrderID == nil {
		s.noteSettleFailure(ctx, in, "order settler", errors.New("order settler not wired or intent has no order"))
		return errors.New("payment: order settler not wired")
	}

	if in.AmountReceivedMicros >= in.AmountExpectedMicros {
		err := s.orders.FulfillPaidOrder(ctx, *in.OrderID, MicrosToUSD(in.AmountExpectedMicros), in.TxHash)
		switch {
		case err == nil:
			// Over-payment: credit the excess (idempotent by ref, so a crash
			// after fulfillment retries safely through this same path).
			if excess := in.AmountReceivedMicros - in.AmountExpectedMicros; excess >= excessCreditMinMicros {
				if _, cerr := s.topUpIdempotent(ctx, in.UserID, MicrosToUSD(excess), methodForNetwork(in.Network), in.ID.Hex()+":excess"); cerr != nil {
					s.noteSettleFailure(ctx, in, "credit excess", cerr)
					return cerr
				}
			}
			if err := s.store.MarkConfirmed(ctx, in.ID, SettlementOrderFulfilled); err != nil {
				s.noteSettleFailure(ctx, in, "mark confirmed", err)
				return err
			}
			s.notify(ctx, in, notification.KindPaymentConfirmed, "Payment confirmed",
				fmt.Sprintf("Your USDT payment of $%.2f was confirmed and your order is on its way.", MicrosToUSD(in.AmountReceivedMicros)))
			return nil

		case errors.Is(err, ErrOrderNotPayable):
			// The order already failed (expiry sweep or a prior underpayment
			// decision won the race) — the money still lands in the wallet.
			if err := s.creditWallet(ctx, in, in.AmountReceivedMicros, in.ID.Hex(), SettlementLate); err != nil {
				return err
			}
			s.notify(ctx, in, notification.KindPaymentConfirmed, "Payment received after the order closed",
				fmt.Sprintf("Your USDT payment of $%.2f arrived after the order was cancelled, so it was added to your wallet.",
					MicrosToUSD(in.AmountReceivedMicros)))
			return nil

		default:
			s.noteSettleFailure(ctx, in, "fulfill order", err)
			return err
		}
	}

	// Underpaid: cancel the order (idempotent — already-failed is fine) and
	// credit what actually arrived.
	if err := s.orders.FailUnpaidOrder(ctx, *in.OrderID, "underpaid"); err != nil {
		s.noteSettleFailure(ctx, in, "fail underpaid order", err)
		return err
	}
	if err := s.creditWallet(ctx, in, in.AmountReceivedMicros, in.ID.Hex(), SettlementUnderpaid); err != nil {
		return err
	}
	s.notify(ctx, in, notification.KindPaymentUnderpaid, "Payment was less than the order total",
		fmt.Sprintf("You sent $%.2f of the $%.2f order total, so the order was cancelled and the amount was added to your wallet.",
			MicrosToUSD(in.AmountReceivedMicros), MicrosToUSD(in.AmountExpectedMicros)))
	return nil
}

// creditWallet credits micros to the user (idempotently, keyed on ref) and
// confirms the intent. A MarkConfirmed failure after a successful credit needs
// no reversal: the retry's credit dedupes on ref and only the mark re-runs.
func (s *Service) creditWallet(ctx context.Context, in *Intent, micros int64, ref, settlement string) error {
	if micros > 0 { // defensive; claims require amount > 0
		if _, err := s.topUpIdempotent(ctx, in.UserID, MicrosToUSD(micros), methodForNetwork(in.Network), ref); err != nil {
			s.noteSettleFailure(ctx, in, "wallet credit", err)
			return err
		}
	}
	if err := s.store.MarkConfirmed(ctx, in.ID, settlement); err != nil {
		s.noteSettleFailure(ctx, in, "mark confirmed", err)
		return err
	}
	return nil
}

// topUpIdempotent credits via wallet.TopUp, treating a duplicate ledger ref as
// success: wallet_transactions carries a unique partial index on (method, ref)
// per on-chain USDT method (usdt_trc20 / usdt_bep20), so a settlement retry
// that already credited hits the duplicate, TopUp's own compensation reverses
// the second balance bump, and we proceed as done.
func (s *Service) topUpIdempotent(ctx context.Context, userID bson.ObjectID, amount float64, method, ref string) (*wallet.WalletTransaction, error) {
	tx, err := s.wallet.TopUp(ctx, userID, amount, method, ref)
	if err != nil && mongo.IsDuplicateKeyError(err) {
		return nil, nil // already credited on a prior attempt
	}
	return tx, err
}

// notify is a best-effort customer notification (inbox row + push).
func (s *Service) notify(ctx context.Context, in *Intent, kind, title, body string) {
	data := map[string]string{
		"intentId": in.ID.Hex(),
		"amount":   fmt.Sprintf("%.2f", MicrosToUSD(in.AmountReceivedMicros)),
	}
	if in.OrderID != nil {
		data["orderId"] = in.OrderID.Hex()
	}
	s.ntf.Notify(ctx, in.UserID, notification.Note{Kind: kind, Title: title, Body: body, Data: data})
}

func (s *Service) noteSettleFailure(ctx context.Context, in *Intent, step string, err error) {
	_ = s.store.IncSettleAttempts(ctx, in.ID)
	slog.Error("payment: settlement step failed — intent stays confirming and will retry",
		"intent", in.ID.Hex(), "purpose", in.Purpose, "step", step, "attempts", in.SettleAttempts+1, "error", err)
}

// ListDeposits returns the shared-address reconciliation queue for admins.
func (s *Service) ListDeposits(ctx context.Context, status string, p pagination.Params) ([]*Deposit, int64, error) {
	return s.store.ListDeposits(ctx, status, p)
}

// AttributeDeposit credits an unmatched shared-address deposit to a customer's
// wallet (v1 reconciliation: wallet credit only — the customer re-orders from
// balance). The atomic unmatched→credited claim blocks two admins crediting
// different users; the wallet credit dedupes on ref "deposit:"+txHash, so a
// retry after a crash can never double-credit.
func (s *Service) AttributeDeposit(ctx context.Context, id, userID bson.ObjectID, adminRef, note string) (*Deposit, error) {
	d, err := s.store.ClaimDepositAttribution(ctx, id, userID, adminRef, note)
	if err != nil {
		return nil, err // ErrConflict = already attributed/ignored (or raced)
	}
	if _, err := s.topUpIdempotent(ctx, userID, MicrosToUSD(d.AmountMicros), methodForNetwork(d.Network), "deposit:"+d.TxHash); err != nil {
		// Put the deposit back in the queue so the admin can retry.
		if rerr := s.store.RevertDepositAttribution(ctx, id); rerr != nil {
			slog.Error("payment: deposit attribution revert failed — deposit stuck credited without a wallet credit",
				"deposit", id.Hex(), "tx", d.TxHash, "error", rerr)
		}
		return nil, err
	}
	s.ntf.Notify(ctx, userID, notification.Note{
		Kind:  notification.KindPaymentConfirmed,
		Title: "Funds added to your wallet",
		Body: fmt.Sprintf("A USDT deposit of $%.2f was matched to your account and added to your wallet.",
			MicrosToUSD(d.AmountMicros)),
		Data: map[string]string{
			"txHash": d.TxHash,
			"amount": fmt.Sprintf("%.2f", MicrosToUSD(d.AmountMicros)),
		},
	})
	return d, nil
}

// IgnoreDeposit marks an unmatched deposit ignored (dust/spam/unknown sender).
func (s *Service) IgnoreDeposit(ctx context.Context, id bson.ObjectID, adminRef, note string) error {
	return s.store.MarkDepositIgnored(ctx, id, adminRef, note)
}
