package user

import (
	"context"
	"log/slog"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// The user module owns account deletion but must not import the modules whose
// state gates or follows it (order/payment/kyc/notification — several of them
// import user). Each dependency is a narrow port here; server.go satisfies
// them with the concrete repositories at wiring time.

// OrderGuard counts a user's in-flight orders (pending/processing) — an
// account with undelivered work cannot be deleted.
type OrderGuard interface {
	CountInFlightByUser(ctx context.Context, userID bson.ObjectID) (int64, error)
}

// PaymentGuard counts a user's open on-chain payment intents
// (pending/confirming) — money possibly en route to the wallet blocks deletion.
type PaymentGuard interface {
	CountOpenByUser(ctx context.Context, userID bson.ObjectID) (int64, error)
}

// TopUpGuard counts a user's pending wallet top-up requests — money already
// paid out-of-band (whish/OMT/cash) awaiting admin approval. Deleting first
// would forfeit it: the approval would credit an anonymized dead account.
type TopUpGuard interface {
	CountPendingForUser(ctx context.Context, userID bson.ObjectID) (int64, error)
}

// KYCPurger permanently removes the user's KYC submission and document photos,
// reporting whether a submission existed. It runs BEFORE the account flips to
// deleted; a failure aborts the deletion.
type KYCPurger interface {
	PurgeByUser(ctx context.Context, userID bson.ObjectID) (bool, error)
}

// DeviceTokenPurger removes the user's registered push device tokens
// (best-effort, after the flip).
type DeviceTokenPurger interface {
	DeleteAllForUser(ctx context.Context, userID bson.ObjectID) (int64, error)
}

// ForfeitLedger writes the wallet-ledger row for a balance zeroed by account
// deletion (forced forfeit, or sub-cent dust). It runs AFTER the flip and is
// best-effort: the flip already zeroed the balance atomically, and the amount
// is also carried by the self-deletion audit entry, so a failed insert is
// logged loudly for reconciliation rather than failing a completed deletion.
type ForfeitLedger interface {
	RecordForfeit(ctx context.Context, userID bson.ObjectID, amount float64) error
}

// DeletionPorts bundles the cross-module dependencies of self-service account
// deletion. All must be wired: DeleteAccount fails closed when any is nil.
type DeletionPorts struct {
	Orders   OrderGuard
	Payments PaymentGuard
	TopUps   TopUpGuard
	KYC      KYCPurger
	Tokens   DeviceTokenPurger
	Ledger   ForfeitLedger
}

// walletDustThreshold is the smallest balance that counts as money for the
// deletion guard. Anything below one cent is rounding dust (3-decimal catalog
// prices charged before cent rounding) that no client can display or spend —
// it must never block deletion. It is zeroed (and ledgered) on the flip.
const walletDustThreshold = 0.01

// DeletionResult summarizes a completed self-deletion for the audit entry
// (no PII): whether a KYC submission was purged, the wallet balance zeroed by
// the flip (forced forfeit, or dust), and how many orders were still in
// flight (only non-zero on a forced deletion — they stay for the operator).
type DeletionResult struct {
	HadKyc           bool
	ForfeitedBalance float64
	InFlightOrders   int64
}

// WithDeletionPorts wires the account-deletion dependencies into the service
// (mirrors WithSettings). Without it DeleteAccount refuses to run (fail-closed)
// so existing call sites and tests keep their behavior.
func WithDeletionPorts(ports DeletionPorts) UserServiceOption {
	return func(s *UserService) { s.deletion = ports }
}

// Deletion guard errors. All are AppErrors so writeAuthError maps them to
// their code + status for the client to render a specific message.
var (
	// ErrWalletNotEmpty refuses deletion while the wallet holds at least a cent
	// — the customer must spend it first, or explicitly retry with force=true
	// to forfeit it ("Delete anyway").
	ErrWalletNotEmpty = &apperrors.AppError{
		Code:    "WALLET_NOT_EMPTY",
		Message: "your wallet still holds balance — spend it before deleting your account",
		Err:     apperrors.ErrConflict,
	}
	// ErrOrdersInFlight refuses deletion while orders are pending/processing
	// (bypassable with force=true — the orders stay for the operator).
	ErrOrdersInFlight = &apperrors.AppError{
		Code:    "ORDERS_IN_FLIGHT",
		Message: "you have orders still being processed — wait for them to complete first",
		Err:     apperrors.ErrConflict,
	}
	// ErrPaymentsPending refuses deletion while USDT payment intents are open.
	ErrPaymentsPending = &apperrors.AppError{
		Code:    "PAYMENTS_PENDING",
		Message: "you have pending payments — wait for them to confirm or expire first",
		Err:     apperrors.ErrConflict,
	}
	// ErrTopUpsPending refuses deletion while a wallet top-up awaits an admin
	// decision — the money is already paid and would be forfeited.
	ErrTopUpsPending = &apperrors.AppError{
		Code:    "TOPUPS_PENDING",
		Message: "you have a wallet top-up awaiting approval — wait for its decision first",
		Err:     apperrors.ErrConflict,
	}
	// ErrAdminAccount refuses self-deletion of admin accounts: admins are
	// removed through the audited admin path, which carries the LAST_ADMIN
	// guard this flow deliberately does not duplicate.
	ErrAdminAccount = &apperrors.AppError{
		Code:    "ADMIN_ACCOUNT",
		Message: "admin accounts cannot be self-deleted",
		Err:     apperrors.ErrForbidden,
	}
)

// DeleteAccount permanently retires the caller's own account: it purges the
// KYC submission + document photos, anonymizes the user document in place
// (soft delete — orders and ledger rows keep resolving), zeroes the wallet,
// and kills every live session. Guards refuse while value is at stake: wallet
// balance (>= 1 cent; sub-cent dust is ignored), in-flight orders, open
// payment intents, or pending top-ups. force=true ("Delete anyway") skips the
// first two — the balance is forfeited (ledgered + audited) and in-flight
// orders stay for the operator against the anonymized account — but money in
// transit (payments, top-ups) and admin/suspended accounts still block.
// Idempotent — deleting an already-deleted (or missing) account succeeds,
// since the JWT outlives the deletion.
func (s *UserService) DeleteAccount(ctx context.Context, id bson.ObjectID, force bool) (DeletionResult, error) {
	var res DeletionResult
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return res, nil // row already gone — nothing left to delete
		}
		return res, err
	}
	if isDeleted(user) {
		// Double-tap after a completed deletion. Re-run the KYC purge anyway: a
		// submission raced in between the original purge and the flip (or was
		// filed by a lingering token before auth.RequireActive shipped) and must
		// not survive the account.
		if s.deletion.KYC != nil {
			if res.HadKyc, err = s.deletion.KYC.PurgeByUser(ctx, id); err != nil {
				return DeletionResult{}, err
			}
		}
		return res, nil
	}
	if user.Role == RoleAdmin {
		return res, ErrAdminAccount
	}
	// A suspended account may not self-delete: SoftDeleteSelf frees the
	// sparse-unique email/phone indexes, so deletion would let a banned person
	// immediately re-register the same phone as a fresh, unsuspended account.
	// They go through support (the web deletion path) instead.
	if isSuspended(user) {
		return res, ErrAccountSuspended
	}

	// Fail closed on missing wiring: without the ports we can guarantee
	// neither the guards nor the KYC purge, so no deletion may proceed.
	if s.deletion.Orders == nil || s.deletion.Payments == nil || s.deletion.TopUps == nil ||
		s.deletion.KYC == nil || s.deletion.Tokens == nil || s.deletion.Ledger == nil {
		slog.Error("user: account deletion requested but deletion ports are not wired", "userId", id.Hex())
		return res, apperrors.ErrInternal
	}

	// Guards, cheapest first — the balance is already on the fetched doc.
	// observed is the ceiling SoftDeleteSelf matches against: any credit that
	// lands after this read misses the flip's filter and refuses the deletion,
	// so a forced delete never forfeits more than the balance seen here.
	observed := user.WalletBalance
	if !force && observed >= walletDustThreshold {
		return res, ErrWalletNotEmpty
	}
	if _, err := s.recheckDeletionGuards(ctx, id, force); err != nil {
		return res, err
	}

	// KYC purge BEFORE the flip: the account must never read deleted while its
	// ID photos still exist. A purge failure aborts — retryable, account intact.
	// Accepted rarity: if a wallet credit races in after this purge,
	// SoftDeleteSelf below refuses and the account survives WITHOUT its KYC —
	// the customer must re-verify to purchase again (photos are already gone).
	res.HadKyc, err = s.deletion.KYC.PurgeByUser(ctx, id)
	if err != nil {
		return DeletionResult{}, err
	}

	// Re-run the count guards: the purge does network blob I/O, easily long
	// enough for a top-up/intent/order to be filed in between — and once the
	// account flips, a pending item settling into the dead wallet is forfeited.
	// (SoftDeleteSelf's balance filter only stops credits landing pre-flip.)
	if res.InFlightOrders, err = s.recheckDeletionGuards(ctx, id, force); err != nil {
		return DeletionResult{HadKyc: res.HadKyc}, err
	}

	// Point of no return: the guarded anonymizing update, which zeroes the
	// wallet in the same document-atomic write. Its balance filter (<= the
	// observed balance) closes the race with a credit landing after the guard.
	before, err := s.repo.SoftDeleteSelf(ctx, id, observed)
	if err != nil {
		if err != apperrors.ErrNotFound {
			return DeletionResult{HadKyc: res.HadKyc}, err
		}
		// Filter miss — re-fetch to disambiguate a concurrent deletion
		// (success) from a concurrent suspension or credit (refuse).
		cur, ferr := s.repo.FindByID(ctx, id)
		switch {
		case ferr == apperrors.ErrNotFound:
			return DeletionResult{HadKyc: res.HadKyc}, nil
		case ferr != nil:
			return DeletionResult{HadKyc: res.HadKyc}, ferr
		case isDeleted(cur):
			return DeletionResult{HadKyc: res.HadKyc}, nil
		case isSuspended(cur):
			return DeletionResult{HadKyc: res.HadKyc}, ErrAccountSuspended
		default:
			return DeletionResult{HadKyc: res.HadKyc}, ErrWalletNotEmpty
		}
	}
	if before != nil && before.WalletBalance > 0 {
		res.ForfeitedBalance = before.WalletBalance
	}

	// Ledger AFTER the flip, best-effort. The flip is the authoritative,
	// atomic balance change; writing the row first would need compensation
	// (deleting a ledger row) whenever the flip then misses on a raced credit.
	// Writing it after can at worst lose the row — and the forfeited amount is
	// still on the self-deletion audit entry and in this error log.
	if res.ForfeitedBalance > 0 {
		if err := s.deletion.Ledger.RecordForfeit(ctx, id, res.ForfeitedBalance); err != nil {
			slog.Error("user: forfeit ledger row failed after self-deletion — reconcile manually",
				"userId", id.Hex(), "forfeited", res.ForfeitedBalance, "error", err)
		}
	}

	// Best-effort cleanup after the flip — never fail the deletion over it.
	// A missed revoke dies with the access-token TTL (Refresh rejects deleted
	// accounts); an orphaned device token pushes to nobody.
	if err := s.refresh.RevokeAllForUser(ctx, id); err != nil {
		slog.Error("user: failed to revoke sessions after self-deletion", "userId", id.Hex(), "error", err)
	}
	if _, err := s.deletion.Tokens.DeleteAllForUser(ctx, id); err != nil {
		slog.Error("user: failed to delete device tokens after self-deletion", "userId", id.Hex(), "error", err)
	}

	slog.Info("user: account self-deleted", "userId", id.Hex(), "hadKyc", res.HadKyc,
		"force", force, "forfeited", res.ForfeitedBalance, "inFlightOrders", res.InFlightOrders)
	return res, nil
}

// recheckDeletionGuards runs the countable deletion guards (in-flight orders,
// open payment intents, pending top-ups). DeleteAccount calls it twice: with
// the initial checks, and again after the KYC purge — whose blob I/O is long
// enough for new items to be filed — so nothing pending can slip past the flip
// and settle money into a dead account. With force the in-flight-order guard
// is skipped (the count is still returned for the audit summary); money in
// transit (payment intents, top-ups) always blocks.
func (s *UserService) recheckDeletionGuards(ctx context.Context, id bson.ObjectID, force bool) (int64, error) {
	inFlight, err := s.deletion.Orders.CountInFlightByUser(ctx, id)
	if err != nil {
		return 0, err
	}
	if inFlight > 0 && !force {
		return inFlight, ErrOrdersInFlight
	}
	open, err := s.deletion.Payments.CountOpenByUser(ctx, id)
	if err != nil {
		return inFlight, err
	}
	if open > 0 {
		return inFlight, ErrPaymentsPending
	}
	pendingTopUps, err := s.deletion.TopUps.CountPendingForUser(ctx, id)
	if err != nil {
		return inFlight, err
	}
	if pendingTopUps > 0 {
		return inFlight, ErrTopUpsPending
	}
	return inFlight, nil
}
