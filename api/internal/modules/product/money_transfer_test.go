package product

import "testing"

func TestValidateMoneyTransfer(t *testing.T) {
	tests := []struct {
		name    string
		ft      FulfillmentType
		in      *MoneyTransfer
		wantErr bool
		// check runs on the normalized result when no error is expected.
		check func(t *testing.T, out *MoneyTransfer)
	}{
		{
			name: "nil passes through",
			ft:   FulfillmentTransfer,
			in:   nil,
			check: func(t *testing.T, out *MoneyTransfer) {
				if out != nil {
					t.Fatalf("want nil, got %+v", out)
				}
			},
		},
		{
			name: "clear sentinel accepted on any type",
			ft:   FulfillmentCode,
			in:   &MoneyTransfer{},
			check: func(t *testing.T, out *MoneyTransfer) {
				if out.hasRates() {
					t.Fatalf("clear sentinel should have no rates, got %+v", out)
				}
			},
		},
		{
			name:    "rejected on non-transfer product",
			ft:      FulfillmentCode,
			in:      &MoneyTransfer{BaseCurrency: "USD", QuoteCurrency: "LBP", BuyRate: 89000},
			wantErr: true,
		},
		{
			name: "buy-only accepted and currencies upper-cased",
			ft:   FulfillmentTransfer,
			in:   &MoneyTransfer{BaseCurrency: " usd ", QuoteCurrency: "lbp", BuyRate: 89000},
			check: func(t *testing.T, out *MoneyTransfer) {
				if out.BaseCurrency != "USD" || out.QuoteCurrency != "LBP" {
					t.Fatalf("currencies not normalized: %+v", out)
				}
				if out.BuyRate != 89000 || out.SellRate != 0 {
					t.Fatalf("rates wrong: %+v", out)
				}
			},
		},
		{
			name: "both rates accepted",
			ft:   FulfillmentTransfer,
			in:   &MoneyTransfer{BaseCurrency: "USD", QuoteCurrency: "LBP", BuyRate: 89000, SellRate: 89500},
			check: func(t *testing.T, out *MoneyTransfer) {
				if out.BuyRate != 89000 || out.SellRate != 89500 {
					t.Fatalf("rates wrong: %+v", out)
				}
			},
		},
		{
			name:    "currencies present but no rate rejected",
			ft:      FulfillmentTransfer,
			in:      &MoneyTransfer{BaseCurrency: "USD", QuoteCurrency: "LBP"},
			wantErr: true,
		},
		{
			name:    "missing quote currency rejected",
			ft:      FulfillmentTransfer,
			in:      &MoneyTransfer{BaseCurrency: "USD", BuyRate: 89000},
			wantErr: true,
		},
		{
			name:    "negative rate rejected",
			ft:      FulfillmentTransfer,
			in:      &MoneyTransfer{BaseCurrency: "USD", QuoteCurrency: "LBP", BuyRate: -1},
			wantErr: true,
		},
		{
			name:    "min above max rejected",
			ft:      FulfillmentTransfer,
			in:      &MoneyTransfer{BaseCurrency: "USD", QuoteCurrency: "LBP", BuyRate: 89000, MinAmount: 100, MaxAmount: 10},
			wantErr: true,
		},
		{
			name: "min/max within bounds accepted",
			ft:   FulfillmentTransfer,
			in:   &MoneyTransfer{BaseCurrency: "USD", QuoteCurrency: "LBP", SellRate: 89500, MinAmount: 5, MaxAmount: 5000},
			check: func(t *testing.T, out *MoneyTransfer) {
				if out.MinAmount != 5 || out.MaxAmount != 5000 {
					t.Fatalf("limits wrong: %+v", out)
				}
			},
		},
		{
			name: "unknown type skips the type guard",
			ft:   "",
			in:   &MoneyTransfer{BaseCurrency: "USD", QuoteCurrency: "LBP", BuyRate: 89000},
			check: func(t *testing.T, out *MoneyTransfer) {
				if !out.hasRates() {
					t.Fatalf("want rates preserved, got %+v", out)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := validateMoneyTransfer(tt.ft, tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil (out=%+v)", out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, out)
			}
		})
	}
}
