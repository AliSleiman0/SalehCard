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

// DeletionPorts bundles the cross-module dependencies of self-service account
// deletion. All four must be wired: DeleteAccount fails closed when any is nil.
type DeletionPorts struct {
	Orders   OrderGuard
	Payments PaymentGuard
	TopUps   TopUpGuard
	KYC      KYCPurger
	Tokens   DeviceTokenPurger
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
	// ErrWalletNotEmpty refuses deletion while the wallet holds balance — the
	// customer must spend it first (deletion never forfeits money).
	ErrWalletNotEmpty = &apperrors.AppError{
		Code:    "WALLET_NOT_EMPTY",
		Message: "your wallet still holds balance — spend it before deleting your account",
		Err:     apperrors.ErrConflict,
	}
	// ErrOrdersInFlight refuses deletion while orders are pending/processing.
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
// (soft delete — orders and ledger rows keep resolving), and kills every live
// session. Guards refuse while value is at stake: wallet balance, in-flight
// orders, or open payment intents. Idempotent — deleting an already-deleted
// (or missing) account succeeds, since the JWT outlives the deletion.
// The returned hadKyc feeds the audit summary (no PII).
func (s *UserService) DeleteAccount(ctx context.Context, id bson.ObjectID) (hadKyc bool, err error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return false, nil // row already gone — nothing left to delete
		}
		return false, err
	}
	if isDeleted(user) {
		// Double-tap after a completed deletion. Re-run the KYC purge anyway: a
		// submission raced in between the original purge and the flip (or was
		// filed by a lingering token before auth.RequireActive shipped) and must
		// not survive the account.
		if s.deletion.KYC != nil {
			if hadKyc, err = s.deletion.KYC.PurgeByUser(ctx, id); err != nil {
				return false, err
			}
		}
		return hadKyc, nil
	}
	if user.Role == RoleAdmin {
		return false, ErrAdminAccount
	}
	// A suspended account may not self-delete: SoftDeleteSelf frees the
	// sparse-unique email/phone indexes, so deletion would let a banned person
	// immediately re-register the same phone as a fresh, unsuspended account.
	// They go through support (the web deletion path) instead.
	if isSuspended(user) {
		return false, ErrAccountSuspended
	}

	// Fail closed on missing wiring: without the ports we can guarantee
	// neither the guards nor the KYC purge, so no deletion may proceed.
	if s.deletion.Orders == nil || s.deletion.Payments == nil || s.deletion.TopUps == nil ||
		s.deletion.KYC == nil || s.deletion.Tokens == nil {
		slog.Error("user: account deletion requested but deletion ports are not wired", "userId", id.Hex())
		return false, apperrors.ErrInternal
	}

	// Guards, cheapest first — the balance is already on the fetched doc.
	if user.WalletBalance > 0 {
		return false, ErrWalletNotEmpty
	}
	if err := s.recheckDeletionGuards(ctx, id); err != nil {
		return false, err
	}

	// KYC purge BEFORE the flip: the account must never read deleted while its
	// ID photos still exist. A purge failure aborts — retryable, account intact.
	// Accepted rarity: if a wallet credit races in after this purge,
	// SoftDeleteSelf below refuses and the account survives WITHOUT its KYC —
	// the customer must re-verify to purchase again (photos are already gone).
	hadKyc, err = s.deletion.KYC.PurgeByUser(ctx, id)
	if err != nil {
		return false, err
	}

	// Re-run the count guards: the purge does network blob I/O, easily long
	// enough for a top-up/intent/order to be filed in between — and once the
	// account flips, a pending item settling into the dead wallet is forfeited.
	// (SoftDeleteSelf's balance filter only stops credits landing pre-flip.)
	if err := s.recheckDeletionGuards(ctx, id); err != nil {
		return hadKyc, err
	}

	// Point of no return: the guarded anonymizing update. Its balance filter
	// closes the race with a wallet credit landing after the guard above.
	if _, err := s.repo.SoftDeleteSelf(ctx, id); err != nil {
		if err != apperrors.ErrNotFound {
			return hadKyc, err
		}
		// Filter miss — re-fetch to disambiguate a concurrent deletion
		// (success) from a concurrent suspension or credit (refuse).
		cur, ferr := s.repo.FindByID(ctx, id)
		switch {
		case ferr == apperrors.ErrNotFound:
			return hadKyc, nil
		case ferr != nil:
			return hadKyc, ferr
		case isDeleted(cur):
			return hadKyc, nil
		case isSuspended(cur):
			return hadKyc, ErrAccountSuspended
		default:
			return hadKyc, ErrWalletNotEmpty
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

	slog.Info("user: account self-deleted", "userId", id.Hex(), "hadKyc", hadKyc)
	return hadKyc, nil
}

// recheckDeletionGuards runs the countable deletion guards (in-flight orders,
// open payment intents, pending top-ups). DeleteAccount calls it twice: with
// the initial checks, and again after the KYC purge — whose blob I/O is long
// enough for new items to be filed — so nothing pending can slip past the flip
// and settle money into a dead account.
func (s *UserService) recheckDeletionGuards(ctx context.Context, id bson.ObjectID) error {
	inFlight, err := s.deletion.Orders.CountInFlightByUser(ctx, id)
	if err != nil {
		return err
	}
	if inFlight > 0 {
		return ErrOrdersInFlight
	}
	open, err := s.deletion.Payments.CountOpenByUser(ctx, id)
	if err != nil {
		return err
	}
	if open > 0 {
		return ErrPaymentsPending
	}
	pendingTopUps, err := s.deletion.TopUps.CountPendingForUser(ctx, id)
	if err != nil {
		return err
	}
	if pendingTopUps > 0 {
		return ErrTopUpsPending
	}
	return nil
}
