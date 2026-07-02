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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
