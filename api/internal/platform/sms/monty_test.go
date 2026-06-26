package sms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func montyTestConfig(baseURL string) MontyConfig {
	return MontyConfig{
		BaseURL:     baseURL,
		Username:    "ozconSer",
		APIID:       "apiid",
		AccessToken: "tok",
		SenderID:    "SalehCard",
	}
}

func TestMontySender_Send_BuildsExpectedRequest(t *testing.T) {
	var gotPath, gotToken string
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Access-Token")
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode(montyResponse{ErrorCode: 0, ID: "abc", MessageCount: 1})
	}))
	defer srv.Close()

	sender, err := newMontySender(montyTestConfig(srv.URL))
	require.NoError(t, err)

	err = sender.Send(context.Background(), "+96170123456", "Your SalehCard verification code is 123456")
	require.NoError(t, err)

	assert.Equal(t, "/API/SendSMS", gotPath)
	assert.Equal(t, "tok", gotToken)
	assert.Equal(t, "ozconSer", gotQuery.Get("username"))
	assert.Equal(t, "apiid", gotQuery.Get("apiId"))
	assert.Equal(t, "True", gotQuery.Get("json"))
	assert.Equal(t, "+96170123456", gotQuery.Get("destination")) // + preserved
	assert.Equal(t, "SalehCard", gotQuery.Get("source"))
	assert.Equal(t, "ozconSer", gotQuery.Get("campaignname")) // defaults to username
	assert.Contains(t, gotQuery.Get("text"), "123456")
}

func TestMontySender_Send_ErrorCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(montyResponse{ErrorCode: -8, Description: "Invalid Source"})
	}))
	defer srv.Close()

	sender, err := newMontySender(montyTestConfig(srv.URL))
	require.NoError(t, err)

	err = sender.Send(context.Background(), "+96170123456", "hi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Source")
}

func TestNewMontySender_RequiresCredentials(t *testing.T) {
	_, err := newMontySender(MontyConfig{SenderID: "SalehCard"}) // missing username/apiId/token
	assert.Error(t, err)
}
