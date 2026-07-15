package wallet

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// fakeMethodStore is an in-memory MethodStore.
type fakeMethodStore struct {
	byID map[bson.ObjectID]*TopUpMethod
}

func newFakeMethodStore() *fakeMethodStore {
	return &fakeMethodStore{byID: map[bson.ObjectID]*TopUpMethod{}}
}

func (f *fakeMethodStore) put(m *TopUpMethod) *TopUpMethod {
	if m.ID.IsZero() {
		m.ID = bson.NewObjectID()
	}
	f.byID[m.ID] = m
	return m
}

func (f *fakeMethodStore) Create(_ context.Context, m *TopUpMethod) error { f.put(m); return nil }
func (f *fakeMethodStore) Update(_ context.Context, _ bson.ObjectID, _ MethodInput) (*TopUpMethod, error) {
	return nil, nil
}
func (f *fakeMethodStore) Delete(_ context.Context, _ bson.ObjectID) error { return nil }
func (f *fakeMethodStore) List(_ context.Context) ([]*TopUpMethod, error) {
	out := []*TopUpMethod{}
	for _, m := range f.byID {
		out = append(out, m)
	}
	return out, nil
}
func (f *fakeMethodStore) ListEnabled(_ context.Context) ([]*TopUpMethod, error) {
	out := []*TopUpMethod{}
	for _, m := range f.byID {
		if m.Enabled {
			out = append(out, m)
		}
	}
	return out, nil
}
func (f *fakeMethodStore) GetByID(_ context.Context, id bson.ObjectID) (*TopUpMethod, error) {
	if m, ok := f.byID[id]; ok {
		return m, nil
	}
	return nil, apperrors.ErrNotFound
}

// bankMethod is a representative method: a required file + a select + optional text.
func bankMethod(store *fakeMethodStore, enabled bool) *TopUpMethod {
	return store.put(&TopUpMethod{
		Name:    "Bank Transfer",
		Enabled: enabled,
		Fields: []MethodField{
			{Key: "receipt", Label: "Transfer receipt", Type: MethodFieldFile, Required: true},
			{Key: "bank", Label: "Bank", Type: MethodFieldSelect, Required: true, Options: []string{"BLOM", "Bankmed"}},
			{Key: "ref", Label: "Reference", Type: MethodFieldText, Required: false},
		},
	})
}

func newMethodSUT() (*WalletService, *fakeMethodStore) {
	repo := &fakeBalanceRepo{balances: map[bson.ObjectID]float64{}}
	methods := newFakeMethodStore()
	return NewWalletService(repo, newFakeTopUpStore()).WithMethods(methods), methods
}

const goodDocURL = "https://x.blob.core.windows.net/product-images/topups/abc.jpg"

func TestCreateTopUpWithMethod(t *testing.T) {
	svc, methods := newMethodSUT()
	uid := bson.NewObjectID()
	m := bankMethod(methods, true)

	// Missing required file → rejected.
	if _, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{
		Amount:   25,
		MethodID: m.ID.Hex(),
		Fields:   []TopUpFieldInput{{Key: "bank", Value: "BLOM"}},
	}); err == nil {
		t.Fatal("missing required file field accepted")
	}

	// Bad select value → rejected.
	if _, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{
		Amount:   25,
		MethodID: m.ID.Hex(),
		Fields: []TopUpFieldInput{
			{Key: "receipt", Value: goodDocURL},
			{Key: "bank", Value: "Nope"},
		},
	}); err == nil {
		t.Fatal("invalid select option accepted")
	}

	// Off-keyspace file URL → rejected.
	if _, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{
		Amount:   25,
		MethodID: m.ID.Hex(),
		Fields: []TopUpFieldInput{
			{Key: "receipt", Value: "https://evil.example/img.png"},
			{Key: "bank", Value: "BLOM"},
		},
	}); err == nil {
		t.Fatal("off-keyspace file URL accepted")
	}

	// Valid submission → fields resolved with server labels, method snapshotted.
	req, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{
		Amount:   25,
		MethodID: m.ID.Hex(),
		Fields: []TopUpFieldInput{
			{Key: "receipt", Value: goodDocURL},
			{Key: "bank", Value: "BLOM"},
			{Key: "ref", Value: "TX-9"},
		},
	})
	if err != nil {
		t.Fatalf("valid method request rejected: %v", err)
	}
	if req.MethodID == nil || *req.MethodID != m.ID || req.MethodName != "Bank Transfer" {
		t.Fatalf("method not snapshotted: %+v", req)
	}
	if len(req.Fields) != 3 {
		t.Fatalf("want 3 fields, got %d: %+v", len(req.Fields), req.Fields)
	}
	for _, f := range req.Fields {
		if f.Key == "receipt" && f.Label != "Transfer receipt" {
			t.Fatalf("label not resolved server-side: %+v", f)
		}
	}
}

func TestCreateTopUpWithDisabledOrUnknownMethod(t *testing.T) {
	svc, methods := newMethodSUT()
	uid := bson.NewObjectID()

	disabled := bankMethod(methods, false)
	if _, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{
		Amount:   25,
		MethodID: disabled.ID.Hex(),
		Fields:   []TopUpFieldInput{{Key: "receipt", Value: goodDocURL}, {Key: "bank", Value: "BLOM"}},
	}); err == nil {
		t.Fatal("disabled method accepted")
	}

	if _, err := svc.CreateTopUpRequest(context.Background(), uid, TopUpInput{
		Amount:   25,
		MethodID: bson.NewObjectID().Hex(),
	}); err == nil {
		t.Fatal("unknown method accepted")
	}
}

func TestMethodInputValidation(t *testing.T) {
	// select without options → rejected.
	if _, err := (MethodInput{
		Name:   "X",
		Fields: []MethodField{{Key: "k", Label: "K", Type: MethodFieldSelect}},
	}).normalizeAndValidate(); err == nil {
		t.Fatal("select without options accepted")
	}
	// duplicate keys → rejected.
	if _, err := (MethodInput{
		Name: "X",
		Fields: []MethodField{
			{Key: "k", Label: "K1", Type: MethodFieldText},
			{Key: "k", Label: "K2", Type: MethodFieldText},
		},
	}).normalizeAndValidate(); err == nil {
		t.Fatal("duplicate field keys accepted")
	}
	// empty name → rejected.
	if _, err := (MethodInput{Name: "  "}).normalizeAndValidate(); err == nil {
		t.Fatal("empty name accepted")
	}
	// valid → options trimmed, fields kept.
	m, err := (MethodInput{
		Name:         "Bank",
		Instructions: "pay here",
		Enabled:      true,
		Fields: []MethodField{
			{Key: "bank", Label: "Bank", Type: MethodFieldSelect, Options: []string{" BLOM ", ""}},
		},
	}).normalizeAndValidate()
	if err != nil {
		t.Fatalf("valid method rejected: %v", err)
	}
	if len(m.Fields) != 1 || len(m.Fields[0].Options) != 1 || m.Fields[0].Options[0] != "BLOM" {
		t.Fatalf("options not normalized: %+v", m.Fields)
	}
}
