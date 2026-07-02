package push

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFCMSender points an FCMSender at a test server with a plain HTTP client
// (no token source) — Send itself does not mint tokens, the oauth2 client does.
func testFCMSender(url string) *FCMSender {
	return &FCMSender{endpoint: url, http: &http.Client{Timeout: 5 * time.Second}}
}

func TestFCMSender_Send_BuildsExpectedRequest(t *testing.T) {
	var gotBody fcmRequest
	var gotContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &gotBody))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := testFCMSender(srv.URL).Send(context.Background(), Message{
		Token: "device-token-1",
		Title: "Order completed",
		Body:  "Your order was delivered — $25.00.",
		Data:  map[string]string{"orderId": "abc123", "kind": "order_completed"},
	})
	require.NoError(t, err)

	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, "device-token-1", gotBody.Message.Token)
	assert.Equal(t, "Order completed", gotBody.Message.Notification.Title)
	assert.Contains(t, gotBody.Message.Notification.Body, "$25.00")
	assert.Equal(t, "abc123", gotBody.Message.Data["orderId"])
	assert.Equal(t, "HIGH", gotBody.Message.Android.Priority)
}

func TestFCMSender_Send_UnregisteredToken(t *testing.T) {
	t.Run("404 maps to ErrUnregistered", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		err := testFCMSender(srv.URL).Send(context.Background(), Message{Token: "stale"})
		assert.True(t, errors.Is(err, ErrUnregistered))
	})

	t.Run("UNREGISTERED error detail maps to ErrUnregistered", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"status":"INVALID_ARGUMENT","details":[{"errorCode":"UNREGISTERED"}]}}`))
		}))
		defer srv.Close()

		err := testFCMSender(srv.URL).Send(context.Background(), Message{Token: "stale"})
		assert.True(t, errors.Is(err, ErrUnregistered))
	})
}

func TestFCMSender_Send_OtherErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"status":"INTERNAL"}}`))
	}))
	defer srv.Close()

	err := testFCMSender(srv.URL).Send(context.Background(), Message{Token: "tok"})
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrUnregistered))
	assert.Contains(t, err.Error(), "500")
}

func TestNewFCMSender_RequiresCredentials(t *testing.T) {
	_, err := newFCMSender(FCMConfig{})
	assert.Error(t, err)
}
