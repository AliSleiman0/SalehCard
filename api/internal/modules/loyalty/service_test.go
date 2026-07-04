package loyalty

import (
	"context"
	"errors"
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/settings"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeConfig struct {
	s   *settings.Settings
	err error
}

func (f fakeConfig) Get(context.Context) (*settings.Settings, error) { return f.s, f.err }

type capturingPoints struct {
	calls     int
	lastDelta int
}

func (c *capturingPoints) AddPoints(_ context.Context, _ bson.ObjectID, delta int) (int, error) {
	c.calls++
	c.lastDelta = delta
	return delta, nil
}

func TestAwarder_Award(t *testing.T) {
	on := &settings.Settings{LoyaltyEnabled: true, LoyaltyEarnUsdPerPoint: 5.0}

	tests := []struct {
		name      string
		cfg       *settings.Settings
		cfgErr    error
		total     float64
		wantCalls int
		wantDelta int
	}{
		{"floors to whole points", on, nil, 27, 1, 5},   // 27/5 = 5.4 -> 5
		{"exact multiple", on, nil, 50, 1, 10},          // 50/5 = 10
		{"below one point earns nothing", on, nil, 3, 0, 0},
		{"disabled", &settings.Settings{LoyaltyEnabled: false, LoyaltyEarnUsdPerPoint: 5}, nil, 100, 0, 0},
		{"zero earn rate", &settings.Settings{LoyaltyEnabled: true, LoyaltyEarnUsdPerPoint: 0}, nil, 100, 0, 0},
		{"settings error is swallowed", nil, errors.New("boom"), 100, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			points := &capturingPoints{}
			a := &Awarder{cfg: fakeConfig{s: tc.cfg, err: tc.cfgErr}, points: points}

			a.Award(context.Background(), bson.NewObjectID(), tc.total)

			if points.calls != tc.wantCalls {
				t.Fatalf("AddPoints calls = %d, want %d", points.calls, tc.wantCalls)
			}
			if tc.wantCalls > 0 && points.lastDelta != tc.wantDelta {
				t.Fatalf("awarded %d points, want %d", points.lastDelta, tc.wantDelta)
			}
		})
	}
}
