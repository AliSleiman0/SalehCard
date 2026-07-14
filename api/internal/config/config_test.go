package config

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "development allows empty secret and origins",
			cfg:     Config{Env: "development"},
			wantErr: false,
		},
		{
			name:    "production requires JWT_SECRET",
			cfg:     Config{Env: "production", AllowedOrigins: "https://example.com"},
			wantErr: true,
		},
		{
			name:    "production requires ALLOWED_ORIGINS",
			cfg:     Config{Env: "production", JWTSecret: "s3cret"},
			wantErr: true,
		},
		{
			name:    "production with secret and origins is valid",
			cfg:     Config{Env: "production", JWTSecret: "s3cret", AllowedOrigins: "https://example.com"},
			wantErr: false,
		},
		{
			name:    "unknown env is treated as strict",
			cfg:     Config{Env: "staging"},
			wantErr: true,
		},
		{
			name:    "both TRC20 modes set refuses even in development",
			cfg:     Config{Env: "development", USDTXPub: "xpub...", USDTAddress: "TAddr"},
			wantErr: true,
		},
		{
			name: "prod trc20 with trongrid + key is valid",
			cfg: Config{Env: "production", JWTSecret: "s", AllowedOrigins: "o",
				USDTAddress: "TAddr", USDTProvider: "trongrid", TronGridAPIKey: "k"},
			wantErr: false,
		},
		{
			name: "prod bep20 stub refused",
			cfg: Config{Env: "production", JWTSecret: "s", AllowedOrigins: "o",
				USDTBEP20Address: "0xabc", USDTBEP20Provider: "stub"},
			wantErr: true,
		},
		{
			name: "prod bep20 etherscan without key refused",
			cfg: Config{Env: "production", JWTSecret: "s", AllowedOrigins: "o",
				USDTBEP20Address: "0xabc", USDTBEP20Provider: "etherscan"},
			wantErr: true,
		},
		{
			name: "prod bep20-only with etherscan + key is valid",
			cfg: Config{Env: "production", JWTSecret: "s", AllowedOrigins: "o",
				USDTBEP20Address: "0xabc", USDTBEP20Provider: "etherscan", EtherscanAPIKey: "k"},
			wantErr: false,
		},
		{
			name: "prod bep20 jsonrpc needs no key",
			cfg: Config{Env: "production", JWTSecret: "s", AllowedOrigins: "o",
				USDTBEP20Address: "0xabc", USDTBEP20Provider: "jsonrpc"},
			wantErr: false,
		},
		{
			name: "prod bep20 unknown provider refused",
			cfg: Config{Env: "production", JWTSecret: "s", AllowedOrigins: "o",
				USDTBEP20Address: "0xabc", USDTBEP20Provider: "jsonrcp"},
			wantErr: true,
		},
		{
			name:    "development allows an unknown bep20 provider",
			cfg:     Config{Env: "development", USDTBEP20Address: "0xabc", USDTBEP20Provider: "jsonrcp"},
			wantErr: false,
		},
		{
			name: "prod both networks valid together",
			cfg: Config{Env: "production", JWTSecret: "s", AllowedOrigins: "o",
				USDTAddress: "TAddr", USDTProvider: "trongrid", TronGridAPIKey: "k",
				USDTBEP20Address: "0xabc", USDTBEP20Provider: "etherscan", EtherscanAPIKey: "k2"},
			wantErr: false,
		},
		{
			name:    "development allows bep20 stub",
			cfg:     Config{Env: "development", USDTBEP20Address: "0xabc", USDTBEP20Provider: "stub"},
			wantErr: false,
		},
		{
			name: "duplicate enabled supplier ids refused even in development",
			cfg: Config{Env: "development", Suppliers: []SupplierConfig{
				{ID: 10, Name: "jentel", Token: "a"},
				{ID: 10, Name: "speedcard", Token: "b"},
			}},
			wantErr: true,
		},
		{
			name: "disabled suppliers may share an id",
			cfg: Config{Env: "development", Suppliers: []SupplierConfig{
				{ID: 10, Name: "jentel", Token: "a"},
				{ID: 10, Name: "speedcard"}, // no token → disabled → harmless
			}},
			wantErr: false,
		},
		{
			name: "mock id colliding with an enabled supplier refused",
			cfg: Config{Env: "development", FulfillmentMock: true, FulfillmentMockID: 10,
				Suppliers: []SupplierConfig{{ID: 10, Name: "jentel", Token: "a"}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadSuppliers(t *testing.T) {
	t.Setenv("SUPPLIER_JENTEL_TOKEN", "tok-j")
	t.Setenv("SUPPLIER_SPEEDCARD_TOKEN", "")
	t.Setenv("SUPPLIER_GIFT4CARD_TOKEN", "")
	t.Setenv("SUPPLIER_SPEEDCARD_ID", "42")

	cfg := Load()
	if len(cfg.Suppliers) != 3 {
		t.Fatalf("Suppliers = %d entries, want 3", len(cfg.Suppliers))
	}
	byName := map[string]SupplierConfig{}
	for _, s := range cfg.Suppliers {
		byName[s.Name] = s
	}
	if s := byName["jentel"]; s.ID != 10 || s.BaseURL != "https://api.jentel-cash.com" || s.Token != "tok-j" {
		t.Errorf("jentel = %+v", s)
	}
	if s := byName["speedcard"]; s.ID != 42 || s.BaseURL != "https://api.speedcard.vip" {
		t.Errorf("speedcard = %+v (ID env override should win)", s)
	}
	if s := byName["gift4card"]; s.ID != 12 || s.BaseURL != "https://api.gift4card.com" {
		t.Errorf("gift4card = %+v", s)
	}

	enabled := cfg.EnabledSuppliers()
	if len(enabled) != 1 || enabled[0].Name != "jentel" {
		t.Errorf("EnabledSuppliers = %+v, want only jentel", enabled)
	}
}
