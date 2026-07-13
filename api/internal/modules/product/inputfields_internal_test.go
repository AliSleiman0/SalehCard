package product

import (
	"errors"
	"strings"
	"testing"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

func fptr(f float64) *float64 { return &f }

func TestSanitizeInputFields(t *testing.T) {
	tests := []struct {
		name    string
		in      []InputField
		want    []InputField // ignored when wantErr is set
		wantErr string       // substring the error message must contain
	}{
		{
			name: "nil passes",
			in:   nil,
			want: nil,
		},
		{
			name: "empty passes",
			in:   []InputField{},
			want: []InputField{},
		},
		{
			name: "valid text amount quantity select pass",
			in: []InputField{
				{Key: "playerId", Type: InputFieldText},
				{Key: "topup", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(100)}},
				{Key: "qty", Type: InputFieldQuantity, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(5000)}},
				{Key: "server", Type: InputFieldSelect, Constraints: &InputFieldConstraints{Options: []string{"EU", "NA"}}},
			},
			want: []InputField{
				{Key: "playerId", Type: InputFieldText},
				{Key: "topup", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(100)}},
				{Key: "qty", Type: InputFieldQuantity, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(5000)}},
				{Key: "server", Type: InputFieldSelect, Constraints: &InputFieldConstraints{Options: []string{"EU", "NA"}}},
			},
		},
		{
			name: "legacy corrupt zero-zero stripped on quantity",
			in:   []InputField{{Key: "qty", Type: InputFieldQuantity, Constraints: &InputFieldConstraints{Min: fptr(0), Max: fptr(0)}}},
			want: []InputField{{Key: "qty", Type: InputFieldQuantity}},
		},
		{
			name: "legacy corrupt zero-zero stripped on amount",
			in:   []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(0), Max: fptr(0)}}},
			want: []InputField{{Key: "amt", Type: InputFieldAmount}},
		},
		{
			name: "options stripped off numeric field",
			in:   []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(9), Options: []string{"stray"}}}},
			want: []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(9)}}},
		},
		{
			name: "constraints stripped off text field",
			in:   []InputField{{Key: "id", Type: InputFieldText, Constraints: &InputFieldConstraints{Min: fptr(1), Options: []string{"x"}}}},
			want: []InputField{{Key: "id", Type: InputFieldText}},
		},
		{
			name: "min and max stripped off select field",
			in:   []InputField{{Key: "srv", Type: InputFieldSelect, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(2), Options: []string{"EU"}}}},
			want: []InputField{{Key: "srv", Type: InputFieldSelect, Constraints: &InputFieldConstraints{Options: []string{"EU"}}}},
		},
		{
			name: "select options trimmed deduped and blanks dropped",
			in:   []InputField{{Key: "srv", Type: InputFieldSelect, Constraints: &InputFieldConstraints{Options: []string{" EU ", "", "NA", "EU", "  "}}}},
			want: []InputField{{Key: "srv", Type: InputFieldSelect, Constraints: &InputFieldConstraints{Options: []string{"EU", "NA"}}}},
		},
		{
			name: "select with no surviving options tolerated as free entry",
			in:   []InputField{{Key: "srv", Type: InputFieldSelect, Constraints: &InputFieldConstraints{Options: []string{"", "  "}}}},
			want: []InputField{{Key: "srv", Type: InputFieldSelect}},
		},
		{
			name: "empty type defaults to text and sheds constraints",
			in:   []InputField{{Key: "id", Constraints: &InputFieldConstraints{Min: fptr(1)}}},
			want: []InputField{{Key: "id", Type: InputFieldText}},
		},
		{
			name: "key is trimmed",
			in:   []InputField{{Key: "  accountId ", Type: InputFieldText}},
			want: []InputField{{Key: "accountId", Type: InputFieldText}},
		},
		{
			name: "one-sided max-only bound accepted",
			in:   []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Max: fptr(100)}}},
			want: []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Max: fptr(100)}}},
		},
		{
			name: "one-sided min-only bound accepted",
			in:   []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(5)}}},
			want: []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(5)}}},
		},
		{
			name: "degenerate equal bounds accepted",
			in:   []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(1)}}},
			want: []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(1), Max: fptr(1)}}},
		},
		{
			name:    "empty key rejected",
			in:      []InputField{{Key: "  ", Type: InputFieldText}},
			wantErr: "needs a key",
		},
		{
			name: "duplicate key rejected",
			in: []InputField{
				{Key: "accountId", Type: InputFieldText},
				{Key: "accountId", Type: InputFieldText},
			},
			wantErr: "must be unique",
		},
		{
			name:    "unknown type rejected",
			in:      []InputField{{Key: "x", Type: "checkbox"}},
			wantErr: "unknown type",
		},
		{
			name:    "negative min rejected",
			in:      []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Min: fptr(-1), Max: fptr(10)}}},
			wantErr: "cannot be negative",
		},
		{
			name:    "negative max rejected",
			in:      []InputField{{Key: "amt", Type: InputFieldAmount, Constraints: &InputFieldConstraints{Max: fptr(-5)}}},
			wantErr: "cannot be negative",
		},
		{
			name:    "min greater than max rejected",
			in:      []InputField{{Key: "qty", Type: InputFieldQuantity, Constraints: &InputFieldConstraints{Min: fptr(10), Max: fptr(2)}}},
			wantErr: "min cannot exceed max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeInputFields(tt.in)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !errors.Is(err, apperrors.ErrBadRequest) {
					t.Errorf("error should wrap ErrBadRequest, got %v", err)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertFieldsEqual(t, tt.want, got)
		})
	}
}

// TestSanitizeInputFields_DoesNotMutateCaller guards the copy-on-write contract:
// normalization must never write through to the caller's backing array.
func TestSanitizeInputFields_DoesNotMutateCaller(t *testing.T) {
	in := []InputField{{Key: "  qty ", Type: InputFieldQuantity, Constraints: &InputFieldConstraints{Min: fptr(0), Max: fptr(0)}}}
	got, err := sanitizeInputFields(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in[0].Key != "  qty " || in[0].Constraints == nil {
		t.Errorf("caller slice was mutated: %+v", in[0])
	}
	if got[0].Key != "qty" || got[0].Constraints != nil {
		t.Errorf("normalized copy wrong: %+v", got[0])
	}
}

func assertFieldsEqual(t *testing.T, want, got []InputField) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("field count: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		w, g := want[i], got[i]
		if w.Key != g.Key || w.Type != g.Type {
			t.Errorf("field %d: want key=%q type=%q, got key=%q type=%q", i, w.Key, w.Type, g.Key, g.Type)
		}
		assertConstraintsEqual(t, i, w.Constraints, g.Constraints)
	}
}

func assertConstraintsEqual(t *testing.T, i int, w, g *InputFieldConstraints) {
	t.Helper()
	if (w == nil) != (g == nil) {
		t.Errorf("field %d constraints: want %+v, got %+v", i, w, g)
		return
	}
	if w == nil {
		return
	}
	if !floatPtrEqual(w.Min, g.Min) || !floatPtrEqual(w.Max, g.Max) {
		t.Errorf("field %d bounds: want min=%v max=%v, got min=%v max=%v", i, w.Min, w.Max, g.Min, g.Max)
	}
	if len(w.Options) != len(g.Options) {
		t.Errorf("field %d options: want %v, got %v", i, w.Options, g.Options)
		return
	}
	for j := range w.Options {
		if w.Options[j] != g.Options[j] {
			t.Errorf("field %d option %d: want %q, got %q", i, j, w.Options[j], g.Options[j])
		}
	}
}

func floatPtrEqual(a, b *float64) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || *a == *b
}
