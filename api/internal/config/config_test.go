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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
