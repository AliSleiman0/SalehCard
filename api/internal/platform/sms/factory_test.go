package sms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_SelectsAdapterByProvider(t *testing.T) {
	monty := MontyConfig{Username: "u", APIID: "a", AccessToken: "tok", SenderID: "SalehCard"}
	twilio := TwilioConfig{AccountSID: "AC1", AuthToken: "tok", From: "+1500"}

	t.Run("empty defaults to log", func(t *testing.T) {
		s, err := New(Config{})
		require.NoError(t, err)
		assert.IsType(t, LogSender{}, s)
	})

	t.Run("log", func(t *testing.T) {
		s, err := New(Config{Provider: "log"})
		require.NoError(t, err)
		assert.IsType(t, LogSender{}, s)
	})

	t.Run("monty", func(t *testing.T) {
		s, err := New(Config{Provider: "monty", Monty: monty})
		require.NoError(t, err)
		assert.IsType(t, &MontySender{}, s)
	})

	t.Run("twilio", func(t *testing.T) {
		s, err := New(Config{Provider: "twilio", Twilio: twilio})
		require.NoError(t, err)
		assert.IsType(t, &TwilioSender{}, s)
	})

	t.Run("monty missing creds errors", func(t *testing.T) {
		_, err := New(Config{Provider: "monty"})
		assert.Error(t, err)
	})

	t.Run("twilio missing creds errors", func(t *testing.T) {
		_, err := New(Config{Provider: "twilio"})
		assert.Error(t, err)
	})

	t.Run("unknown provider errors", func(t *testing.T) {
		_, err := New(Config{Provider: "carrier-pigeon"})
		assert.Error(t, err)
	})
}
