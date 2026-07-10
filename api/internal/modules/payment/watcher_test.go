package payment

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/modules/notification"
	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// fakeReader is a controllable tron.Reader (+TransferLister) keyed by deposit
// address: payments serves the derived FindPayment path, transfers the shared
// ListTransfers path.
type fakeReader struct {
	mu        sync.Mutex
	payments  map[string]*tron.Payment
	transfers map[string][]tron.Payment
	errs      map[string]error
	calls     int
}

func newFakeReader() *fakeReader {
	return &fakeReader{
		payments:  map[string]*tron.Payment{},
		transfers: map[string][]tron.Payment{},
		errs:      map[string]error{},
	}
}

func (f *fakeReader) FindPayment(_ context.Context, w tron.Watch) (*tron.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if err := f.errs[w.Address]; err != nil {
		return nil, err
	}
	return f.payments[w.Address], nil
}

func (f *fakeReader) ListTransfers(_ context.Context, address string, _ time.Time, _ []tron.AmountHint) ([]tron.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if err := f.errs[address]; err != nil {
		return nil, err
	}
	return f.transfers[address], nil
}

func (f *fakeReader) pay(address, txHash string, micros int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payments[address] = &tron.Payment{TxHash: txHash, From: "TSender", AmountMicros: micros, BlockTime: time.Now().UTC()}
}

// transfer appends one shared-address transfer with an explicit block time.
func (f *fakeReader) transfer(address, txHash string, micros int64, blockTime time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.transfers[address] = append(f.transfers[address],
		tron.Payment{TxHash: txHash, From: "TSender", AmountMicros: micros, BlockTime: blockTime})
}

func newWatcherEnv(t *testing.T, cfg Config) (*testEnv, *fakeReader, *Watcher) {
	t.Helper()
	if cfg.XPub == "" {
		cfg.XPub = testXPub(t)
	}
	e := &testEnv{
		store:    newFakeStore(),
		crediter: newFakeCrediter(),
		settler:  newFakeSettler(),
		notifier: &fakeNotifier{},
	}
	reader := newFakeReader()
	e.svc = NewService(e.store, reader, nil, e.crediter, e.notifier, cfg)
	e.svc.SetOrderSettler(e.settler)
	return e, reader, NewWatcher(e.svc, time.Minute)
}

// newSharedWatcherEnv wires a watcher over a shared-address service.
func newSharedWatcherEnv(t *testing.T, cfg Config) (*testEnv, *fakeReader, *Watcher) {
	t.Helper()
	cfg.SharedAddress = testSharedAddr
	e := &testEnv{
		store:    newFakeStore(),
		crediter: newFakeCrediter(),
		settler:  newFakeSettler(),
		notifier: &fakeNotifier{},
	}
	reader := newFakeReader()
	e.svc = NewService(e.store, reader, nil, e.crediter, e.notifier, cfg)
	e.svc.SetOrderSettler(e.settler)
	return e, reader, NewWatcher(e.svc, time.Minute)
}

// unmatchedCount returns how many deposits sit in the reconciliation queue.
func unmatchedCount(t *testing.T, e *testEnv) int {
	t.Helper()
	list, _, err := e.store.ListDeposits(context.Background(), DepositUnmatched, pagination.Params{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	return len(list)
}

func TestWatcherTickScanClaimSettle(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{})
	userID := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), userID, 25, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Nothing on-chain yet: tick is a no-op.
	w.tick(context.Background())
	if got := e.store.get(t, in.ID); got.Status != StatusPending {
		t.Fatalf("status = %s before payment, want pending", got.Status)
	}

	// Transfer appears → claimed + settled in one tick.
	reader.pay(in.Address, "tx-abc", 25_000_000)
	w.tick(context.Background())
	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed || got.TxHash != "tx-abc" || got.Settlement != SettlementWalletTopUp {
		t.Errorf("intent after tick: %+v", got)
	}
	if len(e.crediter.calls) != 1 || e.crediter.calls[0].Amount != 25 {
		t.Errorf("credits: %+v", e.crediter.calls)
	}

	// Further ticks leave the confirmed intent alone.
	w.tick(context.Background())
	if len(e.crediter.calls) != 1 {
		t.Errorf("confirmed intent re-settled: %+v", e.crediter.calls)
	}
}

func TestWatcherExpiryFailsOrderIntent(t *testing.T) {
	e, _, w := newWatcherEnv(t, Config{IntentExpiry: time.Nanosecond})
	userID, orderID := bson.NewObjectID(), bson.NewObjectID()
	in, err := e.svc.CreateOrderIntent(context.Background(), userID, orderID, 50, "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // pass the nanosecond expiry

	w.tick(context.Background())

	if got := e.store.get(t, in.ID); got.Status != StatusExpired {
		t.Fatalf("status = %s, want expired", got.Status)
	}
	if reason := e.settler.failed[orderID]; reason != "payment expired" {
		t.Errorf("order fail reason = %q, want 'payment expired'", reason)
	}
	if kinds := e.notifier.kinds(); len(kinds) != 1 || kinds[0] != notification.KindPaymentExpired {
		t.Errorf("notifications: %v", kinds)
	}
}

func TestWatcherLatePaymentCreditsWallet(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{IntentExpiry: time.Nanosecond})
	userID, orderID := bson.NewObjectID(), bson.NewObjectID()
	in, err := e.svc.CreateOrderIntent(context.Background(), userID, orderID, 50, "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)

	w.tick(context.Background()) // expires the intent + fails the order
	if got := e.store.get(t, in.ID); got.Status != StatusExpired {
		t.Fatalf("precondition: status = %s, want expired", got.Status)
	}

	// The money arrives late (within grace). The order module would refuse
	// fulfillment (already failed) — the payment lands in the wallet.
	e.settler.fulfillErr = ErrOrderNotPayable
	reader.pay(in.Address, "tx-late", 50_000_000)
	w.tick(context.Background())

	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed || got.Settlement != SettlementLate {
		t.Errorf("intent after late payment: status=%s settlement=%q", got.Status, got.Settlement)
	}
	if len(e.crediter.calls) != 1 || e.crediter.calls[0].Amount != 50 {
		t.Errorf("credits: %+v", e.crediter.calls)
	}
}

func TestWatcherReaderErrorDoesNotAbortTick(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	broken, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "k1")
	if err != nil {
		t.Fatal(err)
	}
	healthy, err := e.svc.CreateTopUpIntent(context.Background(), u, 20, "", "k2")
	if err != nil {
		t.Fatal(err)
	}
	reader.errs[broken.Address] = errors.New("trongrid 500")
	reader.pay(healthy.Address, "tx-ok", 20_000_000)

	w.tick(context.Background())

	if got := e.store.get(t, healthy.ID); got.Status != StatusConfirmed {
		t.Errorf("healthy intent status = %s, want confirmed (tick aborted on the broken one?)", got.Status)
	}
	if got := e.store.get(t, broken.ID); got.Status != StatusPending {
		t.Errorf("broken intent status = %s, want pending", got.Status)
	}
}

func TestWatcherRetriesStuckSettlement(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{})
	userID := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), userID, 25, "", "")
	if err != nil {
		t.Fatal(err)
	}
	reader.pay(in.Address, "tx-1", 25_000_000)

	// First tick: claim succeeds, wallet is down → settlement fails.
	e.crediter.failN = 1
	w.tick(context.Background())
	if got := e.store.get(t, in.ID); got.Status != StatusConfirming {
		t.Fatalf("status = %s, want confirming", got.Status)
	}

	// Wallet recovers → the retry pass settles it (no new chain claim needed).
	w.tick(context.Background())
	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed {
		t.Errorf("status after retry tick = %s, want confirmed", got.Status)
	}
	if len(e.crediter.calls) != 1 {
		t.Errorf("credited %d times, want 1", len(e.crediter.calls))
	}
}

func TestWatcherDuplicateTxHashClaimConflict(t *testing.T) {
	e, reader, w := newWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	a, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "ka")
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "kb")
	if err != nil {
		t.Fatal(err)
	}
	// A provider quirk reports the SAME transfer for both intents — the
	// unique txHash guard must credit exactly once.
	reader.pay(a.Address, "tx-same", 10_000_000)
	reader.pay(b.Address, "tx-same", 10_000_000)

	w.tick(context.Background())

	confirmed := 0
	for _, id := range []bson.ObjectID{a.ID, b.ID} {
		if e.store.get(t, id).Status == StatusConfirmed {
			confirmed++
		}
	}
	if confirmed != 1 {
		t.Errorf("confirmed %d intents from one transfer, want 1", confirmed)
	}
	if len(e.crediter.calls) != 1 {
		t.Errorf("credited %d times from one transfer, want 1", len(e.crediter.calls))
	}
}

// --- shared-address scanning --------------------------------------------------

func TestWatcherSharedExactMatchSettles(t *testing.T) {
	e, reader, w := newSharedWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	// Two intents with the SAME base amount — distinguished only by salt.
	a, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "ka")
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "kb")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	reader.transfer(testSharedAddr, "tx-a", a.AmountExpectedMicros, now)
	reader.transfer(testSharedAddr, "tx-b", b.AmountExpectedMicros, now)

	w.tick(context.Background())

	for _, in := range []*Intent{a, b} {
		got := e.store.get(t, in.ID)
		if got.Status != StatusConfirmed || got.Settlement != SettlementWalletTopUp {
			t.Errorf("intent %s after tick: status=%s settlement=%q", in.ID.Hex(), got.Status, got.Settlement)
		}
	}
	if len(e.crediter.calls) != 2 {
		t.Errorf("credited %d times, want 2", len(e.crediter.calls))
	}
	if n := unmatchedCount(t, e); n != 0 {
		t.Errorf("%d unmatched deposits from exact matches, want 0", n)
	}
}

func TestWatcherSharedNonMatchingGoesUnmatched(t *testing.T) {
	e, reader, w := newSharedWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	base := in.AmountExpectedMicros - in.AmountSaltMicros
	reader.transfer(testSharedAddr, "tx-nosalt", base, now)                        // salt stripped
	reader.transfer(testSharedAddr, "tx-over", in.AmountExpectedMicros+50_000, now) // overpaid

	w.tick(context.Background())

	if got := e.store.get(t, in.ID); got.Status != StatusPending {
		t.Errorf("intent status = %s, want pending (nothing matched)", got.Status)
	}
	if len(e.crediter.calls) != 0 {
		t.Errorf("wallet credited from unmatched transfers: %+v", e.crediter.calls)
	}
	if n := unmatchedCount(t, e); n != 2 {
		t.Errorf("%d unmatched deposits, want 2", n)
	}

	// Re-seen on the next tick: the upsert keeps them recorded exactly once.
	w.tick(context.Background())
	if n := unmatchedCount(t, e); n != 2 {
		t.Errorf("%d unmatched deposits after re-scan, want 2", n)
	}
}

func TestWatcherSharedDuplicateEqualTransfer(t *testing.T) {
	e, reader, w := newSharedWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	// Two distinct on-chain transfers of the exact salted amount in one tick —
	// only one may claim the intent; the other is a stranger's coincidence.
	now := time.Now().UTC()
	reader.transfer(testSharedAddr, "tx-first", in.AmountExpectedMicros, now)
	reader.transfer(testSharedAddr, "tx-second", in.AmountExpectedMicros, now)

	w.tick(context.Background())

	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed || got.TxHash != "tx-first" {
		t.Errorf("intent after tick: status=%s tx=%q, want confirmed via tx-first", got.Status, got.TxHash)
	}
	if len(e.crediter.calls) != 1 {
		t.Errorf("credited %d times, want 1", len(e.crediter.calls))
	}
	if n := unmatchedCount(t, e); n != 1 {
		t.Errorf("%d unmatched deposits, want 1 (the duplicate)", n)
	}
}

func TestWatcherSharedLatePaymentCreditsWallet(t *testing.T) {
	e, reader, w := newSharedWatcherEnv(t, Config{IntentExpiry: time.Nanosecond})
	u, orderID := bson.NewObjectID(), bson.NewObjectID()
	in, err := e.svc.CreateOrderIntent(context.Background(), u, orderID, 50, "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)

	w.tick(context.Background()) // expires the intent + fails the order
	if got := e.store.get(t, in.ID); got.Status != StatusExpired {
		t.Fatalf("precondition: status = %s, want expired", got.Status)
	}

	e.settler.fulfillErr = ErrOrderNotPayable
	reader.transfer(testSharedAddr, "tx-late", in.AmountExpectedMicros, time.Now().UTC())
	w.tick(context.Background())

	got := e.store.get(t, in.ID)
	if got.Status != StatusConfirmed || got.Settlement != SettlementLate {
		t.Errorf("late shared payment: status=%s settlement=%q", got.Status, got.Settlement)
	}
	if len(e.crediter.calls) != 1 {
		t.Errorf("credits: %+v", e.crediter.calls)
	}
}

// TestWatcherSharedBlockTimeGuard: a transfer OLDER than the intent must never
// match it, even on an exact amount — it belongs to some earlier consumer of
// that amount slot (or a stranger), so it goes to the queue.
func TestWatcherSharedBlockTimeGuard(t *testing.T) {
	e, reader, w := newSharedWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	reader.transfer(testSharedAddr, "tx-old", in.AmountExpectedMicros, in.CreatedAt.Add(-time.Hour))

	w.tick(context.Background())

	if got := e.store.get(t, in.ID); got.Status != StatusPending {
		t.Errorf("intent claimed by a pre-dating transfer: status = %s", got.Status)
	}
	if n := unmatchedCount(t, e); n != 1 {
		t.Errorf("%d unmatched deposits, want 1", n)
	}
}

func TestWatcherMixedModesInOneTick(t *testing.T) {
	// Service runs in shared mode (new intents share the address), but an
	// open DERIVED intent from before the env flip must keep settling by
	// address polling — the per-intent stamp partitions the scan.
	e, reader, w := newSharedWatcherEnv(t, Config{})
	u := bson.NewObjectID()

	shared, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "ks")
	if err != nil {
		t.Fatal(err)
	}
	derived := &Intent{
		UserID:               u,
		Purpose:              PurposeTopUp,
		Network:              NetworkTRC20,
		Address:              "TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH",
		AddressMode:          AddressModeDerived,
		AmountExpectedMicros: 20_000_000,
		Status:               StatusPending,
		CreatedAt:            time.Now().UTC(),
		ExpiresAt:            time.Now().UTC().Add(time.Hour),
	}
	if err := e.store.Insert(context.Background(), derived); err != nil {
		t.Fatal(err)
	}

	reader.transfer(testSharedAddr, "tx-shared", shared.AmountExpectedMicros, time.Now().UTC())
	reader.pay(derived.Address, "tx-derived", 20_000_000)

	w.tick(context.Background())

	if got := e.store.get(t, shared.ID); got.Status != StatusConfirmed {
		t.Errorf("shared intent status = %s, want confirmed", got.Status)
	}
	if got := e.store.get(t, derived.ID); got.Status != StatusConfirmed {
		t.Errorf("derived intent status = %s, want confirmed", got.Status)
	}
	if len(e.crediter.calls) != 2 {
		t.Errorf("credited %d times, want 2", len(e.crediter.calls))
	}
}

func TestWatcherReleasesSharedSlotsAfterGrace(t *testing.T) {
	e, _, w := newSharedWatcherEnv(t, Config{IntentExpiry: time.Nanosecond, LateGrace: time.Nanosecond})
	u := bson.NewObjectID()
	in, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !e.store.get(t, in.ID).SharedOpen {
		t.Fatal("precondition: sharedOpen not set on creation")
	}
	time.Sleep(time.Millisecond) // pass both expiry and grace

	w.tick(context.Background()) // sweep expires it
	w.tick(context.Background()) // release frees the slot

	got := e.store.get(t, in.ID)
	if got.Status != StatusExpired {
		t.Fatalf("status = %s, want expired", got.Status)
	}
	if got.SharedOpen {
		t.Error("amount slot still reserved after the grace window ended")
	}
}

// --- BEP20 second network ------------------------------------------------

// newDualWatcherEnv wires a watcher over a service with both shared networks
// on; the one fakeReader serves both (transfers are keyed by address, and each
// network has its own address).
func newDualWatcherEnv(t *testing.T, cfg Config) (*testEnv, *fakeReader, *Watcher) {
	t.Helper()
	cfg.SharedAddress = testSharedAddr
	cfg.BEP20SharedAddress = testBEP20Addr
	e := &testEnv{
		store:    newFakeStore(),
		crediter: newFakeCrediter(),
		settler:  newFakeSettler(),
		notifier: &fakeNotifier{},
	}
	reader := newFakeReader()
	e.svc = NewService(e.store, reader, reader, e.crediter, e.notifier, cfg)
	e.svc.SetOrderSettler(e.settler)
	return e, reader, NewWatcher(e.svc, time.Minute)
}

// TestWatcherMixedNetworkTick: one tick settles a TRC20 and a BEP20 shared
// intent independently, each under its own ledger method.
func TestWatcherMixedNetworkTick(t *testing.T) {
	e, reader, w := newDualWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	trc, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, NetworkTRC20, "kt")
	if err != nil {
		t.Fatal(err)
	}
	bep, err := e.svc.CreateTopUpIntent(context.Background(), u, 20, NetworkBEP20, "kb")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	reader.transfer(testSharedAddr, "tx-trc", trc.AmountExpectedMicros, now)
	reader.transfer(testBEP20Addr, "0xtx-bep", bep.AmountExpectedMicros, now)

	w.tick(context.Background())

	gotTrc := e.store.get(t, trc.ID)
	if gotTrc.Status != StatusConfirmed || gotTrc.TxHash != "tx-trc" {
		t.Errorf("trc20 intent after tick: %+v", gotTrc)
	}
	gotBep := e.store.get(t, bep.ID)
	if gotBep.Status != StatusConfirmed || gotBep.TxHash != "0xtx-bep" {
		t.Errorf("bep20 intent after tick: %+v", gotBep)
	}
	methods := map[string]bool{}
	for _, c := range e.crediter.calls {
		methods[c.Method] = true
	}
	if !methods["usdt_trc20"] || !methods["usdt_bep20"] || len(e.crediter.calls) != 2 {
		t.Errorf("credits: %+v", e.crediter.calls)
	}
}

// TestWatcherBEP20UnmatchedDeposit: a BEP20 transfer matching no intent is
// recorded in the reconciliation queue under its own network.
func TestWatcherBEP20UnmatchedDeposit(t *testing.T) {
	e, reader, w := newDualWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	if _, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, NetworkBEP20, ""); err != nil {
		t.Fatal(err)
	}
	reader.transfer(testBEP20Addr, "0xstray", 5_000_000, time.Now().UTC()) // matches nothing

	w.tick(context.Background())

	list, _, err := e.store.ListDeposits(context.Background(), DepositUnmatched, pagination.Params{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Network != NetworkBEP20 || list[0].TxHash != "0xstray" || list[0].ToAddress != testBEP20Addr {
		t.Errorf("unmatched deposits: %+v", list)
	}
}

// TestWatcherSameAmountAcrossNetworks: the same salted total open on BOTH
// networks at once (allowed by the per-network uniqueness) — each chain's
// transfer settles its own intent, never the other's.
func TestWatcherSameAmountAcrossNetworks(t *testing.T) {
	e, reader, w := newDualWatcherEnv(t, Config{})
	u := bson.NewObjectID()
	// Global salt counter: trc20 gets salt 1 (base 10.000000 → 10_000_001),
	// bep20 gets salt 2 (base 9.999999 → 10_000_001) — identical totals.
	trc, err := e.svc.CreateTopUpIntent(context.Background(), u, 10, NetworkTRC20, "kt")
	if err != nil {
		t.Fatal(err)
	}
	bep, err := e.svc.CreateTopUpIntent(context.Background(), u, 9.999999, NetworkBEP20, "kb")
	if err != nil {
		t.Fatal(err)
	}
	if trc.AmountExpectedMicros != bep.AmountExpectedMicros {
		t.Fatalf("precondition: totals differ (%d vs %d)", trc.AmountExpectedMicros, bep.AmountExpectedMicros)
	}
	now := time.Now().UTC()
	reader.transfer(testSharedAddr, "tx-trc", trc.AmountExpectedMicros, now)
	reader.transfer(testBEP20Addr, "0xtx-bep", bep.AmountExpectedMicros, now)

	w.tick(context.Background())

	gotTrc := e.store.get(t, trc.ID)
	gotBep := e.store.get(t, bep.ID)
	if gotTrc.Status != StatusConfirmed || gotTrc.TxHash != "tx-trc" {
		t.Errorf("trc20 intent matched wrong transfer: %+v", gotTrc)
	}
	if gotBep.Status != StatusConfirmed || gotBep.TxHash != "0xtx-bep" {
		t.Errorf("bep20 intent matched wrong transfer: %+v", gotBep)
	}
	if n := unmatchedCount(t, e); n != 0 {
		t.Errorf("unmatched deposits = %d, want 0", n)
	}
}
