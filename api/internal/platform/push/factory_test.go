package push

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_SelectsAdapterByProvider(t *testing.T) {
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

	t.Run("fcm missing creds errors", func(t *testing.T) {
		_, err := New(Config{Provider: "fcm"})
		assert.Error(t, err)
	})

	t.Run("fcm invalid creds errors", func(t *testing.T) {
		_, err := New(Config{Provider: "fcm", FCM: FCMConfig{CredentialsJSON: "not-json"}})
		assert.Error(t, err)
	})

	t.Run("unknown provider errors", func(t *testing.T) {
		_, err := New(Config{Provider: "carrier-pigeon"})
		assert.Error(t, err)
	})
}
