package bridge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// OrderSettler is the slice of the order module the bridge drives on a result.
// Implemented by *order.OrderService and wired post-construction
// (SetOrderSettler) in server.go — order imports bridge to enqueue commands, so
// this side of the loop stays an interface to avoid an import cycle. Both methods
// MUST be idempotent (a result can be reported more than once; the reaper can act
// on the same command state a caller already handled).
type OrderSettler interface {
	// CompleteBridgeOrder completes a processing order whose recharge succeeded,
	// recording transferRef. A no-op when the order is already terminal.
	CompleteBridgeOrder(ctx context.Context, orderID bson.ObjectID, transferRef string) error
	// FlagBridgeOrder records a failed recharge on a processing order WITHOUT
	// changing its status — it stays in the manual admin queue (bridge failures
	// never auto-refund; that is the single money-out path in the order module).
	FlagBridgeOrder(ctx context.Context, orderID bson.ObjectID, reason string) error
}

// OperatorConfig holds the per-operator SMS/USSD templates, shortcodes, and fees
// the device needs — surfaced to the device via the config endpoint so they can
// be recalibrated (operator wording/shortcodes change) without shipping an APK.
type OperatorConfig struct {
	TouchBalanceUSSD      string
	AlfaBalanceUSSD       string
	TouchRechargeTemplate string
	TouchTransferTemplate string
	TouchTransferDest     string
	AlfaRechargeTemplate  string
	AlfaRechargeDest      string
	AlfaTransferTemplate  string
	AlfaTransferDest      string
	TouchMinBalance       float64
	AlfaMinBalance        float64
	TouchMessageFee       float64
	AlfaMessageFee        float64
}

// Config tunes the bridge module. Built from config.Config in RegisterRoutes so
// the bridge package never imports the config package.
type Config struct {
	Enabled           bool
	Stub              bool
	StubDelay         time.Duration
	LeaseTTL          time.Duration
	MaxAttempts       int
	QueueTimeout      time.Duration
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	MaxSMSPerHalfHour int
	SuccessPatterns   []string
	FailurePatterns   []string
	Operators         OperatorConfig
}

// Store is the full persistence surface the service needs — satisfied by
// *MongoStore in production and by an in-memory fake in tests.
type Store interface {
	DeviceStore
	CommandStore
	LogStore
}

// Service orchestrates command creation, leasing, result ingestion, and order
// settlement for the bridge.
type Service struct {
	devices  DeviceStore
	commands CommandStore
	logs     LogStore
	orders   OrderSettler // nil until SetOrderSettler; nil-checked before use
	cfg      Config
}

// NewService constructs the bridge service with sane defaults for zero-valued
// tunables.
func NewService(store Store, cfg Config) *Service {
	if cfg.LeaseTTL <= 0 {
		cfg.LeaseTTL = 3 * time.Minute
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.QueueTimeout <= 0 {
		cfg.QueueTimeout = 30 * time.Minute
	}
	if cfg.StubDelay <= 0 {
		cfg.StubDelay = 10 * time.Second
	}
	return &Service{devices: store, commands: store, logs: store, cfg: cfg}
}

// SetOrderSettler wires the order-completion port. Called once at boot.
func (s *Service) SetOrderSettler(o OrderSettler) { s.orders = o }

// Enabled reports whether bridge fulfillment is turned on. When off, the order
// module falls back to parking recharge orders for manual completion.
func (s *Service) Enabled() bool { return s.cfg.Enabled }

// DispatchOrder enqueues the single recharge command for a placed bridge order.
// The order module has already parked the order (processing) and, for a
// recharge_line, claimed the card code — so by the time the command is visible
// to a device the order is guaranteed to be in a completable state.
func (s *Service) DispatchOrder(ctx context.Context, in DispatchInput) error {
	oid := in.OrderID
	cmd := &Command{
		OrderID:         &oid,
		Provider:        in.Provider,
		RecipientNumber: in.Phone,
		Status:          CommandQueued,
	}
	switch in.Method {
	case "transfer_credit":
		cmd.Type = TypeTransferCredit
		cmd.Amount = in.Amount
	case "recharge_line":
		cmd.Type = TypeRechargeLine
		cmd.CardCode = in.CardCode
	default:
		return fmt.Errorf("bridge: unknown recharge method %q", in.Method)
	}
	return s.commands.CreateCommand(ctx, cmd)
}

// EnqueueBalanceCheck queues a CHECK_BALANCE command per provider a device
// handles (admin-triggered diagnostics).
func (s *Service) EnqueueBalanceCheck(ctx context.Context, d *Device) error {
	for _, p := range d.Providers {
		if err := s.commands.CreateCommand(ctx, &Command{Provider: p, Type: TypeCheckBalance, Status: CommandQueued}); err != nil {
			return err
		}
	}
	return nil
}

// Poll leases up to max queued commands for the device's providers.
func (s *Service) Poll(ctx context.Context, d *Device, max int) ([]*Command, error) {
	if max <= 0 {
		max = 1
	}
	if max > 3 {
		max = 3
	}
	return s.commands.LeaseNext(ctx, d.ID, d.Providers, s.cfg.LeaseTTL, max)
}

// IngestResult records a device's command result idempotently and settles the
// backing order. It returns recorded=true for the first (winning) report and
// recorded=false for a duplicate (already-terminal command) so the device's
// retry loop converges; ErrConflict signals a lost/stale lease.
func (s *Service) IngestResult(ctx context.Context, commandID, deviceID bson.ObjectID, res *CommandResult) (recorded bool, err error) {
	if res.ExecutedAt.IsZero() {
		res.ExecutedAt = time.Now().UTC()
	}
	status := CommandFailed
	if isSuccessCode(res.StatusCode) {
		status = CommandSucceeded
	}
	failReason := ""
	if status == CommandFailed {
		failReason = failureReason(res)
	}

	cmd, err := s.commands.CompleteLeased(ctx, commandID, deviceID, status, res, failReason)
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			// Not holding the lease anymore: an already-terminal command means the
			// device is retrying a report it already delivered — treat as success.
			if existing, ferr := s.commands.FindCommandByID(ctx, commandID); ferr == nil && existing.Status.IsTerminal() {
				return false, nil
			}
			return false, apperrors.ErrConflict
		}
		return false, err
	}

	s.settle(ctx, cmd, status, res)
	return true, nil
}

// settle drives the backing order from a command's terminal outcome. Best effort:
// the command is already finalized, so a settler error is logged, not surfaced —
// the order module's transitions are idempotent and an admin can reconcile.
func (s *Service) settle(ctx context.Context, cmd *Command, status CommandStatus, res *CommandResult) {
	if cmd.OrderID == nil || s.orders == nil {
		return
	}
	if status == CommandSucceeded {
		ref := fmt.Sprintf("bridge:%s", cmd.ID.Hex())
		if err := s.orders.CompleteBridgeOrder(ctx, *cmd.OrderID, ref); err != nil {
			slog.Error("bridge: complete order failed", "command", cmd.ID.Hex(), "order", cmd.OrderID.Hex(), "error", err)
		}
		return
	}
	reason := cmd.FailReason
	if res != nil && res.ErrorMessage != "" {
		reason = res.ErrorMessage
	}
	s.flagOrder(ctx, cmd, reason)
}

// flagOrder marks a command's order as bridge-failed (stays processing for the
// manual queue).
func (s *Service) flagOrder(ctx context.Context, cmd *Command, reason string) {
	if cmd.OrderID == nil || s.orders == nil {
		return
	}
	if reason == "" {
		reason = "mobile recharge failed on the device"
	}
	if err := s.orders.FlagBridgeOrder(ctx, *cmd.OrderID, reason); err != nil {
		slog.Error("bridge: flag order failed", "command", cmd.ID.Hex(), "order", cmd.OrderID.Hex(), "error", err)
	}
}

// Heartbeat records a device's liveness and last-known SIM balances.
func (s *Service) Heartbeat(ctx context.Context, d *Device, hb HeartbeatInput) error {
	now := time.Now().UTC()
	set := bson.D{{Key: "lastSeenAt", Value: now}}
	if hb.TouchBalance != nil {
		set = append(set, bson.E{Key: "touchBalance", Value: *hb.TouchBalance})
	}
	if hb.TouchValidity != "" {
		set = append(set, bson.E{Key: "touchValidity", Value: hb.TouchValidity})
	}
	if hb.AlfaBalance != nil {
		set = append(set, bson.E{Key: "alfaBalance", Value: *hb.AlfaBalance})
	}
	if hb.AlfaValidity != "" {
		set = append(set, bson.E{Key: "alfaValidity", Value: hb.AlfaValidity})
	}
	if hb.AppVersion != "" {
		set = append(set, bson.E{Key: "appVersion", Value: hb.AppVersion})
	}
	return s.devices.UpdateDevice(ctx, d.ID, set)
}

// HeartbeatInput is a device's periodic liveness/state report.
type HeartbeatInput struct {
	TouchBalance  *float64
	TouchValidity string
	AlfaBalance   *float64
	AlfaValidity  string
	AppVersion    string
}

// StoreLogs persists device diagnostic entries (best effort).
func (s *Service) StoreLogs(ctx context.Context, d *Device, entries []LogEntry) error {
	return s.logs.InsertLogs(ctx, d.ID, entries)
}

// BuildConfig assembles the device configuration document: operator templates +
// control knobs + the device's last-known balances (so a reinstall restores
// state). Dates are ISO throughout.
func (s *Service) BuildConfig(d *Device) DeviceConfig {
	pollSec := int(s.cfg.PollInterval / time.Second)
	if pollSec <= 0 {
		pollSec = 5
	}
	hbSec := int(s.cfg.HeartbeatInterval / time.Second)
	if hbSec <= 0 {
		hbSec = 300
	}
	op := s.cfg.Operators
	return DeviceConfig{
		DeviceID:                 d.ID.Hex(),
		PollIntervalSeconds:      pollSec,
		HeartbeatIntervalSeconds: hbSec,
		MaxSMSPerHalfHour:        s.cfg.MaxSMSPerHalfHour,
		SuccessMatchPatterns:     s.cfg.SuccessPatterns,
		FailureMatchPatterns:     s.cfg.FailurePatterns,

		TouchBalanceCheckUssd:             op.TouchBalanceUSSD,
		AlfaBalanceCheckUssd:              op.AlfaBalanceUSSD,
		TouchThirdPartyRechargeTemplate:   op.TouchRechargeTemplate,
		TouchCreditTransferSmsTemplate:    op.TouchTransferTemplate,
		TouchCreditTransferDestination:    op.TouchTransferDest,
		AlfaThirdPartyRechargeSmsTemplate: op.AlfaRechargeTemplate,
		AlfaThirdPartyRechargeDestination: op.AlfaRechargeDest,
		AlfaCreditTransferSmsTemplate:     op.AlfaTransferTemplate,
		AlfaCreditTransferDestination:     op.AlfaTransferDest,
		TouchMinimumAllowedBalance:        op.TouchMinBalance,
		AlfaMinimumAllowedBalance:         op.AlfaMinBalance,
		TouchCreditTransferMessageFee:     op.TouchMessageFee,
		AlfaCreditTransferMessageFee:      op.AlfaMessageFee,

		TouchSimBalance:      valueOr(d.TouchBalance, 0),
		TouchSimValidityDate: d.TouchValidity,
		AlfaSimBalance:       valueOr(d.AlfaBalance, 0),
		AlfaSimValidityDate:  d.AlfaValidity,
	}
}

// ReapTick runs one reaper pass: requeue expired leases, fail commands that no
// device ever claimed, and (dev stub) auto-succeed queued commands so the whole
// completion pipeline is exercised without a real phone.
func (s *Service) ReapTick(ctx context.Context) {
	now := time.Now().UTC()

	failed, err := s.commands.RequeueExpired(ctx, now, s.cfg.MaxAttempts)
	if err != nil {
		slog.Error("bridge: requeue expired failed", "error", err)
	}
	for _, c := range failed {
		s.flagOrder(ctx, c, c.FailReason)
	}

	stale, err := s.commands.FailStaleQueued(ctx, now.Add(-s.cfg.QueueTimeout))
	if err != nil {
		slog.Error("bridge: fail stale queued failed", "error", err)
	}
	for _, c := range stale {
		s.flagOrder(ctx, c, c.FailReason)
	}

	if s.cfg.Stub {
		s.stubSucceed(ctx, now)
	}
}

// stubSucceed (dev only) auto-completes queued commands older than StubDelay by
// leasing them to a synthetic device id and pushing a success result through the
// real ingest path — so a dev with no phone still sees orders complete.
func (s *Service) stubSucceed(ctx context.Context, now time.Time) {
	stale, err := s.commands.ListStaleStub(ctx, now.Add(-s.cfg.StubDelay), 50)
	if err != nil {
		slog.Error("bridge: stub list failed", "error", err)
		return
	}
	for _, c := range stale {
		leased, lerr := s.commands.LeaseNext(ctx, stubDeviceID, []string{c.Provider}, s.cfg.LeaseTTL, 1)
		if lerr != nil || len(leased) == 0 {
			continue
		}
		lc := leased[0]
		code := 2000
		switch lc.Type {
		case TypeRechargeLine:
			if lc.Provider == "alfa" {
				code = 4000
			} else {
				code = 3000
			}
		case TypeCheckBalance:
			code = 5000
		}
		if _, err := s.IngestResult(ctx, lc.ID, stubDeviceID, &CommandResult{StatusCode: code, RawReply: "STUB: auto-succeeded", ExecutedAt: now}); err != nil {
			slog.Warn("bridge: stub ingest failed", "command", lc.ID.Hex(), "error", err)
		}
	}
}

// stubDeviceID is the synthetic lessee for stub-mode auto-completion.
var stubDeviceID = func() bson.ObjectID {
	id, _ := bson.ObjectIDFromHex("000000000000000000000001")
	return id
}()

// failureReason produces a short human reason from a result code / message.
func failureReason(res *CommandResult) string {
	if res.ErrorMessage != "" {
		return res.ErrorMessage
	}
	if isPartialCode(res.StatusCode) {
		return "recharge only partially completed"
	}
	return fmt.Sprintf("device reported failure code %d", res.StatusCode)
}

func valueOr(p *float64, def float64) float64 {
	if p != nil {
		return *p
	}
	return def
}
