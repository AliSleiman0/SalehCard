package bridge

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// --- fake store ------------------------------------------------------------

type fakeStore struct {
	cmds map[bson.ObjectID]*Command
}

func newFakeStore() *fakeStore { return &fakeStore{cmds: map[bson.ObjectID]*Command{}} }

func (f *fakeStore) CreateCommand(_ context.Context, c *Command) error {
	if c.ID.IsZero() {
		c.ID = bson.NewObjectID()
	}
	if c.Status == "" {
		c.Status = CommandQueued
	}
	cp := *c
	f.cmds[c.ID] = &cp
	return nil
}
func (f *fakeStore) FindCommandByID(_ context.Context, id bson.ObjectID) (*Command, error) {
	if c, ok := f.cmds[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakeStore) FindCommandsByOrder(context.Context, bson.ObjectID) ([]*Command, error) {
	return nil, nil
}
func (f *fakeStore) LeaseNext(_ context.Context, deviceID bson.ObjectID, providers []string, ttl time.Duration, max int) ([]*Command, error) {
	out := []*Command{}
	for _, c := range f.cmds {
		if len(out) >= max {
			break
		}
		if c.Status != CommandQueued {
			continue
		}
		if len(providers) > 0 && !contains(providers, c.Provider) {
			continue
		}
		exp := time.Now().Add(ttl)
		c.Status = CommandLeased
		c.DeviceID = &deviceID
		c.LeaseExpiresAt = &exp
		c.Attempts++
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
}
func (f *fakeStore) CompleteLeased(_ context.Context, id, deviceID bson.ObjectID, status CommandStatus, res *CommandResult, failReason string) (*Command, error) {
	c, ok := f.cmds[id]
	if !ok || c.Status != CommandLeased || c.DeviceID == nil || *c.DeviceID != deviceID {
		return nil, apperrors.ErrConflict
	}
	c.Status = status
	c.Result = res
	c.FailReason = failReason
	c.LeaseExpiresAt = nil
	cp := *c
	return &cp, nil
}
func (f *fakeStore) RequeueExpired(_ context.Context, now time.Time, maxAttempts int) ([]*Command, error) {
	failed := []*Command{}
	for _, c := range f.cmds {
		if c.Status == CommandLeased && c.LeaseExpiresAt != nil && c.LeaseExpiresAt.Before(now) {
			if c.Attempts >= maxAttempts {
				c.Status = CommandFailed
				c.FailReason = "recharge attempts exhausted"
				cp := *c
				failed = append(failed, &cp)
			} else {
				c.Status = CommandQueued
				c.DeviceID = nil
				c.LeaseExpiresAt = nil
			}
		}
	}
	return failed, nil
}
func (f *fakeStore) FailStaleQueued(_ context.Context, cutoff time.Time) ([]*Command, error) {
	failed := []*Command{}
	for _, c := range f.cmds {
		if c.Status == CommandQueued && c.CreatedAt.Before(cutoff) {
			c.Status = CommandFailed
			c.FailReason = "recharge timed out waiting for a device"
			cp := *c
			failed = append(failed, &cp)
		}
	}
	return failed, nil
}
func (f *fakeStore) ListStaleStub(context.Context, time.Time, int) ([]*Command, error) {
	return nil, nil
}
func (f *fakeStore) RetryCommand(context.Context, bson.ObjectID) (*Command, error) { return nil, nil }
func (f *fakeStore) CancelCommand(context.Context, bson.ObjectID) (*Command, error) {
	return nil, nil
}
func (f *fakeStore) ListCommands(context.Context, CommandFilter, pagination.Params) ([]*Command, int64, error) {
	return nil, 0, nil
}

// device + log no-ops
func (f *fakeStore) CreateDevice(context.Context, *Device) error { return nil }
func (f *fakeStore) FindDeviceByID(context.Context, bson.ObjectID) (*Device, error) {
	return nil, apperrors.ErrNotFound
}
func (f *fakeStore) FindDeviceByTokenHash(context.Context, string) (*Device, error) {
	return nil, apperrors.ErrNotFound
}
func (f *fakeStore) ListDevices(context.Context) ([]*Device, error)              { return nil, nil }
func (f *fakeStore) UpdateDevice(context.Context, bson.ObjectID, bson.D) error   { return nil }
func (f *fakeStore) DeleteDevice(context.Context, bson.ObjectID) error           { return nil }
func (f *fakeStore) InsertLogs(context.Context, bson.ObjectID, []LogEntry) error { return nil }

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

// --- fake settler ----------------------------------------------------------

type fakeSettler struct {
	completed map[bson.ObjectID]string
	flagged   map[bson.ObjectID]string
}

func newFakeSettler() *fakeSettler {
	return &fakeSettler{completed: map[bson.ObjectID]string{}, flagged: map[bson.ObjectID]string{}}
}
func (f *fakeSettler) CompleteBridgeOrder(_ context.Context, orderID bson.ObjectID, ref string) error {
	f.completed[orderID] = ref
	return nil
}
func (f *fakeSettler) FlagBridgeOrder(_ context.Context, orderID bson.ObjectID, reason string) error {
	f.flagged[orderID] = reason
	return nil
}

// --- helpers ---------------------------------------------------------------

func newTestSvc() (*Service, *fakeStore, *fakeSettler) {
	store := newFakeStore()
	settler := newFakeSettler()
	svc := NewService(store, Config{Enabled: true, LeaseTTL: time.Minute, MaxAttempts: 3, QueueTimeout: 30 * time.Minute})
	svc.SetOrderSettler(settler)
	return svc, store, settler
}

func leaseOne(t *testing.T, svc *Service, dev *Device) *Command {
	t.Helper()
	cmds, err := svc.Poll(context.Background(), dev, 1)
	if err != nil || len(cmds) != 1 {
		t.Fatalf("lease: err=%v n=%d", err, len(cmds))
	}
	return cmds[0]
}

// --- tests -----------------------------------------------------------------

func TestIngestResult_SuccessCompletesOrderOnce(t *testing.T) {
	svc, _, settler := newTestSvc()
	dev := &Device{ID: bson.NewObjectID(), Providers: []string{"touch"}, Enabled: true}
	orderID := bson.NewObjectID()
	if err := svc.DispatchOrder(context.Background(), DispatchInput{OrderID: orderID, Provider: "touch", Method: "transfer_credit", Phone: "71123456", Amount: f64(5)}); err != nil {
		t.Fatal(err)
	}
	cmd := leaseOne(t, svc, dev)

	recorded, err := svc.IngestResult(context.Background(), cmd.ID, dev.ID, &CommandResult{StatusCode: 2000})
	if err != nil || !recorded {
		t.Fatalf("first ingest: recorded=%v err=%v", recorded, err)
	}
	if _, ok := settler.completed[orderID]; !ok {
		t.Fatal("order was not completed")
	}

	// A duplicate report of the same (now terminal) command converges to duplicate.
	recorded, err = svc.IngestResult(context.Background(), cmd.ID, dev.ID, &CommandResult{StatusCode: 2000})
	if err != nil || recorded {
		t.Fatalf("duplicate ingest: recorded=%v err=%v (want false,nil)", recorded, err)
	}
}

func TestIngestResult_FailureFlagsOrderNoComplete(t *testing.T) {
	svc, _, settler := newTestSvc()
	dev := &Device{ID: bson.NewObjectID(), Providers: []string{"alfa"}, Enabled: true}
	orderID := bson.NewObjectID()
	_ = svc.DispatchOrder(context.Background(), DispatchInput{OrderID: orderID, Provider: "alfa", Method: "recharge_line", Phone: "71123456", CardCode: "ABC123"})
	cmd := leaseOne(t, svc, dev)

	recorded, err := svc.IngestResult(context.Background(), cmd.ID, dev.ID, &CommandResult{StatusCode: 4022, ErrorMessage: "rejected by provider"})
	if err != nil || !recorded {
		t.Fatalf("ingest: recorded=%v err=%v", recorded, err)
	}
	if _, ok := settler.completed[orderID]; ok {
		t.Fatal("failed recharge must NOT complete the order")
	}
	if reason := settler.flagged[orderID]; reason == "" {
		t.Fatal("failed recharge must flag the order")
	}
}

func TestIngestResult_PartialTransferIsFailure(t *testing.T) {
	svc, _, settler := newTestSvc()
	dev := &Device{ID: bson.NewObjectID(), Providers: []string{"touch"}, Enabled: true}
	orderID := bson.NewObjectID()
	_ = svc.DispatchOrder(context.Background(), DispatchInput{OrderID: orderID, Provider: "touch", Method: "transfer_credit", Phone: "71123456", Amount: f64(9)})
	cmd := leaseOne(t, svc, dev)

	_, _ = svc.IngestResult(context.Background(), cmd.ID, dev.ID, &CommandResult{StatusCode: 2001})
	if _, ok := settler.completed[orderID]; ok {
		t.Fatal("partial transfer must not complete the order")
	}
	if _, ok := settler.flagged[orderID]; !ok {
		t.Fatal("partial transfer must flag the order")
	}
}

func TestIngestResult_StaleLeaseConflicts(t *testing.T) {
	svc, _, _ := newTestSvc()
	dev := &Device{ID: bson.NewObjectID(), Providers: []string{"touch"}, Enabled: true}
	_ = svc.DispatchOrder(context.Background(), DispatchInput{OrderID: bson.NewObjectID(), Provider: "touch", Method: "transfer_credit", Phone: "71123456", Amount: f64(5)})
	cmd := leaseOne(t, svc, dev)
	// A different device reporting the same command holds no lease → conflict.
	other := bson.NewObjectID()
	if _, err := svc.IngestResult(context.Background(), cmd.ID, other, &CommandResult{StatusCode: 2000}); err == nil {
		t.Fatal("expected conflict for a non-holder device")
	}
}

func TestReapTick_ExhaustedLeaseFlagsOrder(t *testing.T) {
	svc, store, settler := newTestSvc()
	dev := &Device{ID: bson.NewObjectID(), Providers: []string{"touch"}, Enabled: true}
	orderID := bson.NewObjectID()
	_ = svc.DispatchOrder(context.Background(), DispatchInput{OrderID: orderID, Provider: "touch", Method: "transfer_credit", Phone: "71123456", Amount: f64(5)})

	// Lease-then-expire each round; after MaxAttempts the reaper fails + flags it.
	for i := 0; i < 5; i++ {
		cmds, _ := svc.Poll(context.Background(), dev, 1)
		if len(cmds) == 1 {
			past := time.Now().Add(-time.Hour)
			store.cmds[cmds[0].ID].LeaseExpiresAt = &past
		}
		svc.ReapTick(context.Background())
	}
	if _, ok := settler.flagged[orderID]; !ok {
		t.Fatal("an exhausted command must flag its order")
	}
}

func TestProviderFilterOnLease(t *testing.T) {
	svc, _, _ := newTestSvc()
	touchDev := &Device{ID: bson.NewObjectID(), Providers: []string{"touch"}, Enabled: true}
	_ = svc.DispatchOrder(context.Background(), DispatchInput{OrderID: bson.NewObjectID(), Provider: "alfa", Method: "transfer_credit", Phone: "71123456", Amount: f64(5)})
	cmds, err := svc.Poll(context.Background(), touchDev, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 0 {
		t.Fatalf("a touch-only device must not lease an alfa command, got %d", len(cmds))
	}
}

func TestTokenRoundTrip(t *testing.T) {
	plain, hash, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	if HashToken(plain) != hash {
		t.Fatal("hash mismatch")
	}
	if HashToken("bd_different") == hash {
		t.Fatal("different token produced same hash")
	}
	if len(plain) < 10 || plain[:3] != "bd_" {
		t.Fatalf("unexpected token shape: %q", plain)
	}
}

func TestSuccessCodeClassification(t *testing.T) {
	for _, c := range []int{1000, 2000, 3000, 4000, 5000} {
		if !isSuccessCode(c) {
			t.Errorf("code %d should be success", c)
		}
	}
	for _, c := range []int{2001, 2023, 4022, 9002, 0} {
		if isSuccessCode(c) {
			t.Errorf("code %d should be failure", c)
		}
	}
	if !isPartialCode(2001) {
		t.Error("2001 is the partial code")
	}
}

func f64(v float64) *float64 { return &v }
