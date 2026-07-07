package payment

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/modules/wallet"
	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
	"github.com/AliSleiman0/salehcard/api/internal/platform/whish"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
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

// Config tunes the payment module.
type Config struct {
	// XPub is the watch-only account key deposit addresses derive from.
	XPub string
	// IntentExpiry is how long a customer has to pay (default 30m).
	IntentExpiry time.Duration
	// LateGrace is how long after expiry the watcher keeps scanning an unpaid
	// address so a late transfer still lands as a wallet credit (default 7d).
	LateGrace time.Duration
	// MaxPending caps a user's open intents (default 3).
	MaxPending int

	// WebhookBaseURL is the public host Whish reaches for server callbacks
	// (e.g. a dev tunnel, or the prod API origin). Whish's sandbox cannot reach
	// localhost.
	WebhookBaseURL string
	// WhishSuccessRedirectURL / WhishFailureRedirectURL are where the customer's
	// browser lands after the hosted page. Optional (a native client can rely on
	// its own poll instead).
	WhishSuccessRedirectURL string
	WhishFailureRedirectURL string
	// WhishIntentExpiry is the customer's Whish payment window (default 30m).
	WhishIntentExpiry time.Duration
}

// Service orchestrates intent creation and settlement for both providers.
type Service struct {
	store  Store
	reader tron.Reader     // USDT chain reader; nil when USDT is off
	whish  whish.Provider  // Whish gateway; nil when Whish is off
	tokens *Tokens         // HMAC signer for Whish callbacks; nil when Whish is off
	wallet topUpCrediter
	orders OrderSettler // nil until SetOrderSettler; nil-checked in settle
	ntf    notification.Notifier
	cfg    Config
}

// NewService constructs the payment service. reader enables USDT; whish+tokens
// enable Whish; either, both, or neither may be configured.
func NewService(store Store, reader tron.Reader, wlt topUpCrediter, ntf notification.Notifier, cfg Config) *Service {
	if cfg.IntentExpiry <= 0 {
		cfg.IntentExpiry = defaultIntentExpiry
	}
	if cfg.LateGrace <= 0 {
		cfg.LateGrace = defaultLateGrace
	}
	if cfg.MaxPending <= 0 {
		cfg.MaxPending = defaultMaxPending
	}
	if cfg.WhishIntentExpiry <= 0 {
		cfg.WhishIntentExpiry = defaultIntentExpiry
	}
	if ntf == nil {
		ntf = notification.Nop{}
	}
	return &Service{store: store, reader: reader, wallet: wlt, ntf: ntf, cfg: cfg}
}

// SetWhish wires the Whish provider + HMAC tokens (redirect payments). Called at
// boot when Whish is configured. Both must be non-nil for WhishEnabled.
func (s *Service) SetWhish(p whish.Provider, tokens *Tokens) {
	s.whish = p
	s.tokens = tokens
}

// SetOrderSettler wires the order-fulfillment port. Called once at boot after
// both services exist (order needs payment first for intent creation).
func (s *Service) SetOrderSettler(o OrderSettler) { s.orders = o }

// Enabled reports whether on-chain USDT payments are configured.
func (s *Service) Enabled() bool { return s.cfg.XPub != "" && s.reader != nil }

// WhishEnabled reports whether Whish redirect payments are configured.
func (s *Service) WhishEnabled() bool { return s.whish != nil && s.tokens != nil }

// ExpiryMinutes exposes the configured window for the client config endpoint.
func (s *Service) ExpiryMinutes() int { return int(s.cfg.IntentExpiry / time.Minute) }

// CreateTopUpIntent opens an on-chain USDT top-up intent for the user.
func (s *Service) CreateTopUpIntent(ctx context.Context, userID bson.ObjectID, amountUSD float64, idemKey string) (*Intent, error) {
	return s.createIntent(ctx, &Intent{
		UserID:   userID,
		Purpose:  PurposeTopUp,
		Provider: ProviderUSDT,
	}, amountUSD, idemKey)
}

// CreateOrderIntent opens the on-chain USDT payment intent for a just-placed
// order. PlaceOrder's own Idempotency-Key dedup covers retries at the order
// level, so the intent derives its key from the order id (one live intent per
// order).
func (s *Service) CreateOrderIntent(ctx context.Context, userID, orderID bson.ObjectID, amountUSD float64) (*Intent, error) {
	oid := orderID
	return s.createIntent(ctx, &Intent{
		UserID:   userID,
		Purpose:  PurposeOrder,
		OrderID:  &oid,
		Provider: ProviderUSDT,
	}, amountUSD, "order:"+orderID.Hex())
}

// CreateWhishTopUpIntent opens a Whish redirect top-up intent for the user.
func (s *Service) CreateWhishTopUpIntent(ctx context.Context, userID bson.ObjectID, amountUSD float64, idemKey string) (*Intent, error) {
	return s.createIntent(ctx, &Intent{
		UserID:   userID,
		Purpose:  PurposeTopUp,
		Provider: ProviderWhish,
	}, amountUSD, idemKey)
}

// CreateWhishOrderIntent opens the Whish payment intent for a just-placed order
// (key derived from the order id, like the USDT path).
func (s *Service) CreateWhishOrderIntent(ctx context.Context, userID, orderID bson.ObjectID, amountUSD float64) (*Intent, error) {
	oid := orderID
	return s.createIntent(ctx, &Intent{
		UserID:   userID,
		Purpose:  PurposeOrder,
		OrderID:  &oid,
		Provider: ProviderWhish,
	}, amountUSD, "whish-order:"+orderID.Hex())
}

func (s *Service) createIntent(ctx context.Context, in *Intent, amountUSD float64, idemKey string) (*Intent, error) {
	isWhish := in.Provider == ProviderWhish
	if (isWhish && !s.WhishEnabled()) || (!isWhish && !s.Enabled()) {
		return nil, &apperrors.AppError{
			Code:    "PAYMENT_METHOD_UNAVAILABLE",
			Message: "this payment method is not available right now",
			Err:     apperrors.ErrBadRequest,
		}
	}
	if amountUSD <= 0 {
		return nil, &apperrors.AppError{Code: "BAD_REQUEST", Message: "amount must be positive", Err: apperrors.ErrBadRequest}
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

	if isWhish {
		in.ExpiresAt = now.Add(s.cfg.WhishIntentExpiry)
		eid, err := newExternalID()
		if err != nil {
			return nil, fmt.Errorf("payment: external id: %w", err)
		}
		in.ExternalID = eid
	} else {
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
		in.Network = NetworkTRC20
		in.Address = address
		in.DerivationIndex = index
		in.ExpiresAt = now.Add(s.cfg.IntentExpiry)
	}

	if err := s.store.Insert(ctx, in); err != nil {
		if errors.Is(err, apperrors.ErrConflict) && idemKey != "" {
			// A concurrent request won the idempotency race; return the winner.
			if existing, ferr := s.store.GetByIdempotencyKey(ctx, in.UserID, idemKey); ferr == nil {
				return existing, nil
			}
		}
		return nil, err
	}

	if isWhish {
		if err := s.initiateWhish(ctx, in, amountUSD); err != nil {
			return nil, err
		}
	}
	return in, nil
}

// initiateWhish opens the hosted Whish page for a freshly-inserted pending
// intent and persists the redirect URL. On a gateway error it marks the intent
// failed (so its idempotency key isn't permanently blocked) and returns an
// upstream error.
func (s *Service) initiateWhish(ctx context.Context, in *Intent, amountUSD float64) error {
	token := s.tokens.SignExternalID(in.ExternalID)
	res, err := s.whish.Initiate(ctx, whish.InitiateInput{
		Amount:             amountUSD,
		Currency:           "USD",
		Invoice:            s.whishInvoice(in),
		ExternalID:         in.ExternalID,
		SuccessCallbackURL: s.callbackURL(true, in.ExternalID, token),
		FailureCallbackURL: s.callbackURL(false, in.ExternalID, token),
		SuccessRedirectURL: s.cfg.WhishSuccessRedirectURL,
		FailureRedirectURL: s.cfg.WhishFailureRedirectURL,
	})
	if err != nil {
		_ = s.store.MarkFailed(ctx, in.ID, "whish initiate failed")
		slog.Error("payment: whish initiate failed", "intent", in.ID.Hex(), "error", err)
		return &apperrors.AppError{
			Code:    "PAYMENT_METHOD_UNAVAILABLE",
			Message: "could not start the Whish payment — please try again",
			Err:     apperrors.ErrBadRequest,
		}
	}
	if err := s.store.UpdateAfterInitiate(ctx, in.ID, res.RedirectURL, res.ProviderRef); err != nil {
		return err
	}
	in.RedirectURL = res.RedirectURL
	in.ProviderRef = res.ProviderRef
	return nil
}

// whishInvoice builds the human-facing description Whish shows on the page.
func (s *Service) whishInvoice(in *Intent) string {
	if in.Purpose == PurposeOrder && in.OrderID != nil {
		return "SalehCard order " + in.OrderID.Hex()
	}
	return "SalehCard wallet top-up"
}

// callbackURL builds the public webhook URL Whish fires server-to-server,
// carrying the external id + its HMAC token (the token authenticates the call).
func (s *Service) callbackURL(success bool, externalID int64, token string) string {
	outcome := "failure"
	if success {
		outcome = "success"
	}
	q := url.Values{}
	q.Set("externalId", strconv.FormatInt(externalID, 10))
	q.Set("token", token)
	return s.cfg.WebhookBaseURL + "/api/v1/payments/webhooks/whish/" + outcome + "?" + q.Encode()
}

// newExternalID returns a positive int64 unique-per-request id: millisecond
// timestamp << 16 | 16 bits of crypto entropy (ported from LACPA).
func newExternalID() (int64, error) {
	base := time.Now().UnixMilli() << 16
	jitter, err := rand.Int(rand.Reader, big.NewInt(1<<16))
	if err != nil {
		return 0, err
	}
	id := base | jitter.Int64()
	if id < 0 {
		id = -id
	}
	return id, nil
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

// ErrInvalidCallback is a malformed / unauthenticated Whish callback (bad
// external id or a token that fails HMAC verification). The handler maps it to
// 400 so Whish does not retry (a genuine retry can't fix a bad signature).
var ErrInvalidCallback = errors.New("payment: invalid whish callback")

// HandleCallback processes a Whish server-to-server callback. The callback is
// unsigned, so the HMAC token in the query string is the authentication; then
// the gateway status is re-polled as the source of truth (the browser can land
// on the failure redirect even when the payment actually succeeded — never
// trust which URL fired). Idempotent: a re-delivered callback on a terminal
// intent is a no-op, and a callback during a stuck settlement resumes it.
func (s *Service) HandleCallback(ctx context.Context, externalIDStr, token string) error {
	if !s.WhishEnabled() {
		return errors.New("payment: whish not enabled")
	}
	externalID, err := strconv.ParseInt(externalIDStr, 10, 64)
	if err != nil {
		return ErrInvalidCallback
	}
	if !s.tokens.VerifyExternalID(externalID, token) {
		return ErrInvalidCallback
	}
	in, err := s.store.GetByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return ErrInvalidCallback
		}
		return err
	}

	switch in.Status {
	case StatusConfirmed, StatusExpired, StatusFailed:
		return nil // terminal — idempotent re-ack
	case StatusConfirming:
		return s.settle(ctx, in) // resume a settlement interrupted earlier
	}

	// Pending: re-poll the gateway for ground truth.
	st, err := s.whish.GetStatus(ctx, whish.StatusQuery{ExternalID: externalID, Currency: "USD"})
	if err != nil {
		return fmt.Errorf("payment: whish status re-poll: %w", err)
	}
	switch st.Status {
	case whish.CollectStatusSuccess:
		return s.confirmWhish(ctx, in, st.PayerPhone)
	case whish.CollectStatusFailed:
		return s.failWhish(ctx, in, "whish_failed")
	case whish.CollectStatusPending:
		return nil // not resolved yet; a later callback / the sweep settles it
	default:
		return fmt.Errorf("payment: unknown whish status %q", st.Status)
	}
}

// confirmWhish claims a pending Whish intent (received = expected) and settles
// it. A concurrent claim (ErrConflict) is resolved by settling the current
// state, keeping the whole flow idempotent.
func (s *Service) confirmWhish(ctx context.Context, in *Intent, payerPhone string) error {
	claimed, err := s.store.ClaimWhishConfirmed(ctx, in.ID, in.AmountExpectedMicros, payerPhone)
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			cur, gerr := s.store.GetByID(ctx, in.ID)
			if gerr != nil {
				return gerr
			}
			if cur.Status == StatusConfirming {
				return s.settle(ctx, cur)
			}
			return nil // already confirmed/terminal
		}
		return err
	}
	return s.settle(ctx, claimed)
}

// failWhish marks a pending Whish intent failed and cancels the order behind it
// (best-effort). Idempotent: an already-moved intent is a no-op.
func (s *Service) failWhish(ctx context.Context, in *Intent, reason string) error {
	if err := s.store.MarkFailed(ctx, in.ID, reason); err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			return nil // already moved on (concurrent callback / sweep)
		}
		return err
	}
	if in.Purpose == PurposeOrder && in.OrderID != nil && s.orders != nil {
		if err := s.orders.FailUnpaidOrder(ctx, *in.OrderID, reason); err != nil {
			slog.Error("payment: fail whish order failed", "intent", in.ID.Hex(), "order", in.OrderID.Hex(), "error", err)
		}
	}
	s.notify(ctx, in, notification.KindPaymentExpired, "Payment was not completed",
		"Your Whish payment did not go through. No money was taken; you can try again.")
	return nil
}

// SweepWhishExpired reconciles pending Whish intents past their expiry: it
// re-polls the gateway (guarding a lost-callback-but-paid race) and settles a
// late success, records a failure, or expires an abandoned intent (failing the
// order behind it). Called on a ticker while Whish is enabled.
func (s *Service) SweepWhishExpired(ctx context.Context) {
	if !s.WhishEnabled() {
		return
	}
	now := time.Now().UTC()
	candidates, err := s.store.ListWhishExpiryCandidates(ctx, now, perTickLimit)
	if err != nil {
		slog.Error("payment: list whish expiry candidates failed", "error", err)
		return
	}
	for _, in := range candidates {
		if ctx.Err() != nil {
			return
		}
		st, err := s.whish.GetStatus(ctx, whish.StatusQuery{ExternalID: in.ExternalID, Currency: "USD"})
		if err != nil {
			slog.Warn("payment: whish reconcile status failed", "intent", in.ID.Hex(), "error", err)
			continue // leave pending; retry next tick
		}
		switch st.Status {
		case whish.CollectStatusSuccess:
			if err := s.confirmWhish(ctx, in, st.PayerPhone); err != nil {
				slog.Error("payment: whish reconcile settle failed", "intent", in.ID.Hex(), "error", err)
			}
		case whish.CollectStatusFailed:
			if err := s.failWhish(ctx, in, "whish_failed"); err != nil {
				slog.Error("payment: whish reconcile fail failed", "intent", in.ID.Hex(), "error", err)
			}
		default: // pending → genuinely abandoned; expire it
			s.expireWhish(ctx, in, now)
		}
	}
}

// expireWhish flips an abandoned pending Whish intent to expired and fails the
// order behind it (best-effort).
func (s *Service) expireWhish(ctx context.Context, in *Intent, now time.Time) {
	expired, err := s.store.ExpireOne(ctx, in.ID, now)
	if err != nil {
		if !errors.Is(err, apperrors.ErrConflict) {
			slog.Error("payment: expire whish intent failed", "intent", in.ID.Hex(), "error", err)
		}
		return
	}
	if expired.Purpose == PurposeOrder && expired.OrderID != nil && s.orders != nil {
		if err := s.orders.FailUnpaidOrder(ctx, *expired.OrderID, "payment expired"); err != nil {
			slog.Error("payment: fail expired whish order failed", "intent", expired.ID.Hex(), "order", expired.OrderID.Hex(), "error", err)
		}
	}
	s.notify(ctx, expired, notification.KindPaymentExpired, "Payment window expired",
		"Your Whish payment window expired before the payment completed.")
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
			fmt.Sprintf("Your payment of $%.2f was confirmed and added to your wallet.", MicrosToUSD(in.AmountReceivedMicros)))
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
				if _, cerr := s.topUpIdempotent(ctx, in.UserID, MicrosToUSD(excess), walletMethod(in), in.ID.Hex()+":excess"); cerr != nil {
					s.noteSettleFailure(ctx, in, "credit excess", cerr)
					return cerr
				}
			}
			if err := s.store.MarkConfirmed(ctx, in.ID, SettlementOrderFulfilled); err != nil {
				s.noteSettleFailure(ctx, in, "mark confirmed", err)
				return err
			}
			s.notify(ctx, in, notification.KindPaymentConfirmed, "Payment confirmed",
				fmt.Sprintf("Your payment of $%.2f was confirmed and your order is on its way.", MicrosToUSD(in.AmountReceivedMicros)))
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
		if _, err := s.topUpIdempotent(ctx, in.UserID, MicrosToUSD(micros), walletMethod(in), ref); err != nil {
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
// for settlement rows (usdt_trc20 / whish), so a settlement retry that already
// credited hits the duplicate, TopUp's own compensation reverses the second
// balance bump, and we proceed as done.
func (s *Service) topUpIdempotent(ctx context.Context, userID bson.ObjectID, amount float64, method, ref string) (*wallet.WalletTransaction, error) {
	tx, err := s.wallet.TopUp(ctx, userID, amount, method, ref)
	if err != nil && mongo.IsDuplicateKeyError(err) {
		return nil, nil // already credited on a prior attempt
	}
	return tx, err
}

// walletMethod is the wallet-ledger method string for an intent's provider (the
// ref-scoped idempotency key of its settlement credit). Kept in sync with the
// wallet_transactions (method, ref) partial-unique index filter.
func walletMethod(in *Intent) string {
	if ProviderOf(in) == ProviderWhish {
		return "whish"
	}
	return "usdt_trc20"
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
