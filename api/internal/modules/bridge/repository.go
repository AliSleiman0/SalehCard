package bridge

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// reapLimit caps how many stale commands one reaper pass processes, so a tick
// stays bounded; the rest are picked up next pass (oldest first).
const reapLimit = 200

// CommandFilter narrows the admin command listing. Zero-valued fields are ignored.
type CommandFilter struct {
	Status   string
	Provider string
	OrderID  *bson.ObjectID
}

// DeviceStore persists bridge devices.
type DeviceStore interface {
	CreateDevice(ctx context.Context, d *Device) error
	FindDeviceByID(ctx context.Context, id bson.ObjectID) (*Device, error)
	FindDeviceByTokenHash(ctx context.Context, hash string) (*Device, error)
	ListDevices(ctx context.Context) ([]*Device, error)
	UpdateDevice(ctx context.Context, id bson.ObjectID, set bson.D) error
	DeleteDevice(ctx context.Context, id bson.ObjectID) error
}

// CommandStore persists bridge commands with the atomic lease/complete claims
// that make polling and result-reporting safe under redelivery and scale-out.
type CommandStore interface {
	CreateCommand(ctx context.Context, c *Command) error
	FindCommandByID(ctx context.Context, id bson.ObjectID) (*Command, error)
	FindCommandsByOrder(ctx context.Context, orderID bson.ObjectID) ([]*Command, error)
	// LeaseNext atomically claims up to max queued commands for the given
	// provider set, moving each to leased with a fresh lease window and bumping
	// its attempt counter. Oldest first. Returns the leased commands (possibly
	// empty).
	LeaseNext(ctx context.Context, deviceID bson.ObjectID, providers []string, ttl time.Duration, max int) ([]*Command, error)
	// CompleteLeased atomically finalizes a command the device holds a lease on
	// (status leased AND deviceId matches) to a terminal status, attaching the
	// result. ErrConflict when the lease is no longer held (already terminal,
	// reassigned, or expired) — the caller resolves duplicate vs stale.
	CompleteLeased(ctx context.Context, id, deviceID bson.ObjectID, status CommandStatus, res *CommandResult, failReason string) (*Command, error)
	// RequeueExpired moves leased commands whose lease has expired back to queued
	// (attempts < max) or to failed (attempts exhausted). Returns the newly-failed
	// commands so the caller can flag their orders.
	RequeueExpired(ctx context.Context, now time.Time, maxAttempts int) ([]*Command, error)
	// FailStaleQueued fails commands that have sat queued past the cutoff (no
	// device ever picked them up). Returns them for order-flagging.
	FailStaleQueued(ctx context.Context, cutoff time.Time) ([]*Command, error)
	// ListStaleStub returns queued commands older than cutoff (dev stub mode).
	ListStaleStub(ctx context.Context, cutoff time.Time, limit int) ([]*Command, error)
	RetryCommand(ctx context.Context, id bson.ObjectID) (*Command, error)
	CancelCommand(ctx context.Context, id bson.ObjectID) (*Command, error)
	ListCommands(ctx context.Context, f CommandFilter, p pagination.Params) ([]*Command, int64, error)
}

// LogStore persists device diagnostic logs (TTL-expired).
type LogStore interface {
	InsertLogs(ctx context.Context, deviceID bson.ObjectID, entries []LogEntry) error
}

// LogEntry is one device diagnostic line.
type LogEntry struct {
	DeviceID  bson.ObjectID `bson:"deviceId"`
	Level     string        `bson:"level"`
	ErrorCode *int          `bson:"errorCode,omitempty"`
	Message   string        `bson:"message"`
	At        time.Time     `bson:"at"`
	CreatedAt time.Time     `bson:"createdAt"` // TTL anchor
}

// MongoStore is the MongoDB-backed implementation of all three stores over
// bridge_devices / bridge_commands / bridge_logs.
type MongoStore struct {
	devices  *mongo.Collection
	commands *mongo.Collection
	logs     *mongo.Collection
}

// NewMongoStore constructs a MongoStore.
func NewMongoStore(db *mongo.Database) *MongoStore {
	return &MongoStore{
		devices:  db.Collection("bridge_devices"),
		commands: db.Collection("bridge_commands"),
		logs:     db.Collection("bridge_logs"),
	}
}

// EnsureIndexes creates the bridge indexes: a unique token hash for O(1) device
// auth, command indexes for the lease scan / order lookup / admin listing, and a
// 30-day TTL on device logs.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if _, err := db.Collection("bridge_devices").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)},
	}); err != nil {
		return err
	}
	if _, err := db.Collection("bridge_commands").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "provider", Value: 1}, {Key: "createdAt", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "leaseExpiresAt", Value: 1}}},
		{Keys: bson.D{{Key: "orderId", Value: 1}}, Options: options.Index().SetPartialFilterExpression(bson.D{{Key: "orderId", Value: bson.D{{Key: "$exists", Value: true}}}})},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}); err != nil {
		return err
	}
	_, err := db.Collection("bridge_logs").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "createdAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(30 * 24 * 3600),
	})
	return err
}

// --- devices ---------------------------------------------------------------

func (s *MongoStore) CreateDevice(ctx context.Context, d *Device) error {
	if d.ID.IsZero() {
		d.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now
	if d.Providers == nil {
		d.Providers = []string{}
	}
	if _, err := s.devices.InsertOne(ctx, d); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrConflict
		}
		return err
	}
	return nil
}

func (s *MongoStore) FindDeviceByID(ctx context.Context, id bson.ObjectID) (*Device, error) {
	return s.oneDevice(ctx, bson.D{{Key: "_id", Value: id}})
}

func (s *MongoStore) FindDeviceByTokenHash(ctx context.Context, hash string) (*Device, error) {
	return s.oneDevice(ctx, bson.D{{Key: "tokenHash", Value: hash}})
}

func (s *MongoStore) oneDevice(ctx context.Context, filter bson.D) (*Device, error) {
	var d Device
	if err := s.devices.FindOne(ctx, filter).Decode(&d); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (s *MongoStore) ListDevices(ctx context.Context) ([]*Device, error) {
	cur, err := s.devices.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*Device{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *MongoStore) UpdateDevice(ctx context.Context, id bson.ObjectID, set bson.D) error {
	set = append(set, bson.E{Key: "updatedAt", Value: time.Now().UTC()})
	res, err := s.devices.UpdateOne(ctx, bson.D{{Key: "_id", Value: id}}, bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (s *MongoStore) DeleteDevice(ctx context.Context, id bson.ObjectID) error {
	res, err := s.devices.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

// --- commands --------------------------------------------------------------

func (s *MongoStore) CreateCommand(ctx context.Context, c *Command) error {
	if c.ID.IsZero() {
		c.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = CommandQueued
	}
	_, err := s.commands.InsertOne(ctx, c)
	return err
}

func (s *MongoStore) FindCommandByID(ctx context.Context, id bson.ObjectID) (*Command, error) {
	var c Command
	if err := s.commands.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&c); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (s *MongoStore) FindCommandsByOrder(ctx context.Context, orderID bson.ObjectID) ([]*Command, error) {
	cur, err := s.commands.Find(ctx, bson.D{{Key: "orderId", Value: orderID}}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*Command{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// LeaseNext claims up to max queued commands one at a time (each an atomic
// FindOneAndUpdate, oldest first), so two devices polling concurrently can never
// lease the same command.
func (s *MongoStore) LeaseNext(ctx context.Context, deviceID bson.ObjectID, providers []string, ttl time.Duration, max int) ([]*Command, error) {
	out := make([]*Command, 0, max)
	for i := 0; i < max; i++ {
		match := bson.D{{Key: "status", Value: CommandQueued}}
		if len(providers) > 0 {
			match = append(match, bson.E{Key: "provider", Value: bson.D{{Key: "$in", Value: providers}}})
		}
		now := time.Now().UTC()
		var c Command
		err := s.commands.FindOneAndUpdate(ctx,
			match,
			bson.D{
				{Key: "$set", Value: bson.D{
					{Key: "status", Value: CommandLeased},
					{Key: "deviceId", Value: deviceID},
					{Key: "leaseExpiresAt", Value: now.Add(ttl)},
					{Key: "updatedAt", Value: now},
				}},
				{Key: "$inc", Value: bson.D{{Key: "attempts", Value: 1}}},
			},
			options.FindOneAndUpdate().
				SetSort(bson.D{{Key: "createdAt", Value: 1}}).
				SetReturnDocument(options.After),
		).Decode(&c)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				break // nothing left to lease
			}
			return out, err
		}
		out = append(out, &c)
	}
	return out, nil
}

// CompleteLeased finalizes a device-held command (see CommandStore).
func (s *MongoStore) CompleteLeased(ctx context.Context, id, deviceID bson.ObjectID, status CommandStatus, res *CommandResult, failReason string) (*Command, error) {
	set := bson.D{
		{Key: "status", Value: status},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}
	if res != nil {
		set = append(set, bson.E{Key: "result", Value: res})
	}
	if failReason != "" {
		set = append(set, bson.E{Key: "failReason", Value: failReason})
	}
	var c Command
	err := s.commands.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: id},
			{Key: "status", Value: CommandLeased},
			{Key: "deviceId", Value: deviceID},
		},
		bson.D{
			{Key: "$set", Value: set},
			{Key: "$unset", Value: bson.D{{Key: "leaseExpiresAt", Value: ""}}},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &c, nil
}

// RequeueExpired handles lease expiry (see CommandStore). Exhausted commands are
// failed and returned; the rest are moved back to queued for another device.
func (s *MongoStore) RequeueExpired(ctx context.Context, now time.Time, maxAttempts int) ([]*Command, error) {
	failed := make([]*Command, 0)
	// Fail the exhausted ones first (atomic per-command), then bulk-requeue the rest.
	for i := 0; i < reapLimit; i++ {
		var c Command
		err := s.commands.FindOneAndUpdate(ctx,
			bson.D{
				{Key: "status", Value: CommandLeased},
				{Key: "leaseExpiresAt", Value: bson.D{{Key: "$lt", Value: now}}},
				{Key: "attempts", Value: bson.D{{Key: "$gte", Value: maxAttempts}}},
			},
			bson.D{
				{Key: "$set", Value: bson.D{
					{Key: "status", Value: CommandFailed},
					{Key: "failReason", Value: "recharge attempts exhausted"},
					{Key: "updatedAt", Value: now},
				}},
				{Key: "$unset", Value: bson.D{{Key: "leaseExpiresAt", Value: ""}, {Key: "deviceId", Value: ""}}},
			},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(&c)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				break
			}
			return failed, err
		}
		failed = append(failed, &c)
	}
	// Requeue the still-retriable expired leases in one shot.
	if _, err := s.commands.UpdateMany(ctx,
		bson.D{
			{Key: "status", Value: CommandLeased},
			{Key: "leaseExpiresAt", Value: bson.D{{Key: "$lt", Value: now}}},
		},
		bson.D{
			{Key: "$set", Value: bson.D{{Key: "status", Value: CommandQueued}, {Key: "updatedAt", Value: now}}},
			{Key: "$unset", Value: bson.D{{Key: "leaseExpiresAt", Value: ""}, {Key: "deviceId", Value: ""}}},
		},
	); err != nil {
		return failed, err
	}
	return failed, nil
}

// FailStaleQueued fails commands that were never picked up in time.
func (s *MongoStore) FailStaleQueued(ctx context.Context, cutoff time.Time) ([]*Command, error) {
	failed := make([]*Command, 0)
	for i := 0; i < reapLimit; i++ {
		var c Command
		err := s.commands.FindOneAndUpdate(ctx,
			bson.D{
				{Key: "status", Value: CommandQueued},
				{Key: "createdAt", Value: bson.D{{Key: "$lt", Value: cutoff}}},
			},
			bson.D{{Key: "$set", Value: bson.D{
				{Key: "status", Value: CommandFailed},
				{Key: "failReason", Value: "recharge timed out waiting for a device"},
				{Key: "updatedAt", Value: time.Now().UTC()},
			}}},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(&c)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				break
			}
			return failed, err
		}
		failed = append(failed, &c)
	}
	return failed, nil
}

// ListStaleStub returns queued commands older than cutoff (dev stub auto-succeed).
func (s *MongoStore) ListStaleStub(ctx context.Context, cutoff time.Time, limit int) ([]*Command, error) {
	cur, err := s.commands.Find(ctx,
		bson.D{{Key: "status", Value: CommandQueued}, {Key: "createdAt", Value: bson.D{{Key: "$lt", Value: cutoff}}}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}).SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*Command{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RetryCommand re-queues a failed/cancelled command, resetting its attempt count
// (an admin explicitly chose to try again). The SAME card code is preserved.
func (s *MongoStore) RetryCommand(ctx context.Context, id bson.ObjectID) (*Command, error) {
	var c Command
	err := s.commands.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: id},
			{Key: "status", Value: bson.D{{Key: "$in", Value: []CommandStatus{CommandFailed, CommandCancelled}}}},
		},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "status", Value: CommandQueued},
				{Key: "attempts", Value: 0},
				{Key: "updatedAt", Value: time.Now().UTC()},
			}},
			{Key: "$unset", Value: bson.D{{Key: "leaseExpiresAt", Value: ""}, {Key: "deviceId", Value: ""}, {Key: "failReason", Value: ""}, {Key: "result", Value: ""}}},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			if _, ferr := s.FindCommandByID(ctx, id); ferr != nil {
				return nil, apperrors.ErrNotFound
			}
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &c, nil
}

// CancelCommand cancels a queued or leased command.
func (s *MongoStore) CancelCommand(ctx context.Context, id bson.ObjectID) (*Command, error) {
	var c Command
	err := s.commands.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: id},
			{Key: "status", Value: bson.D{{Key: "$in", Value: []CommandStatus{CommandQueued, CommandLeased}}}},
		},
		bson.D{
			{Key: "$set", Value: bson.D{{Key: "status", Value: CommandCancelled}, {Key: "updatedAt", Value: time.Now().UTC()}}},
			{Key: "$unset", Value: bson.D{{Key: "leaseExpiresAt", Value: ""}, {Key: "deviceId", Value: ""}}},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			if _, ferr := s.FindCommandByID(ctx, id); ferr != nil {
				return nil, apperrors.ErrNotFound
			}
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return &c, nil
}

// ListCommands returns the paginated admin command view, newest first.
func (s *MongoStore) ListCommands(ctx context.Context, f CommandFilter, p pagination.Params) ([]*Command, int64, error) {
	filter := bson.D{}
	if v := strings.TrimSpace(f.Status); v != "" {
		filter = append(filter, bson.E{Key: "status", Value: v})
	}
	if v := strings.TrimSpace(f.Provider); v != "" {
		filter = append(filter, bson.E{Key: "provider", Value: v})
	}
	if f.OrderID != nil {
		filter = append(filter, bson.E{Key: "orderId", Value: *f.OrderID})
	}
	total, err := s.commands.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := s.commands.Find(ctx, filter, options.Find().
		SetSkip(pagination.Skip(p)).
		SetLimit(int64(p.Limit)).
		SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := []*Command{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// InsertLogs stores device diagnostic entries (best effort; TTL-expired).
func (s *MongoStore) InsertLogs(ctx context.Context, deviceID bson.ObjectID, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	now := time.Now().UTC()
	docs := make([]any, len(entries))
	for i := range entries {
		entries[i].DeviceID = deviceID
		entries[i].CreatedAt = now
		if entries[i].At.IsZero() {
			entries[i].At = now
		}
		docs[i] = entries[i]
	}
	_, err := s.logs.InsertMany(ctx, docs)
	return err
}
