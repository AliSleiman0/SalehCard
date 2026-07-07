package whish

import "errors"

// Sandbox / recommended defaults.
const (
	// DefaultSandboxBaseURL is the Whish sandbox API root. Production is
	// https://api.whish.money/itel-service/api.
	DefaultSandboxBaseURL = "https://api.sandbox.whish.money/itel-service/api"
	// DefaultUserAgent is the UA Whish recommends callers send.
	DefaultUserAgent = "Whish/1.0 (https://whish.money; support@whish.money)"
)

// ClientConfig holds the Whish HTTP adapter's credentials and endpoint. Channel,
// Secret, and WebsiteURL are per-merchant and required; BaseURL/UserAgent fall
// back to sandbox/recommended defaults.
type ClientConfig struct {
	BaseURL    string
	Channel    string
	Secret     string
	WebsiteURL string
	UserAgent  string
}

// validate fills defaults and rejects missing credentials so New can degrade
// the provider gracefully rather than 500 on the first payment.
func (c *ClientConfig) validate() error {
	if c.BaseURL == "" {
		c.BaseURL = DefaultSandboxBaseURL
	}
	if c.UserAgent == "" {
		c.UserAgent = DefaultUserAgent
	}
	if c.Channel == "" || c.Secret == "" || c.WebsiteURL == "" {
		return errors.New("whish: WHISH_CHANNEL, WHISH_SECRET, and WHISH_WEBSITE_URL must be set")
	}
	return nil
}
