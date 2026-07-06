package bridge

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

// DeviceConfig is the configuration document served to a device. Field names are
// the camelCase keys the Android client's Gson parser expects (a superset of the
// legacy ConfigurationDTO plus the alfa-balance pair and control knobs).
type DeviceConfig struct {
	DeviceID                 string   `json:"deviceId"`
	PollIntervalSeconds      int      `json:"pollIntervalSeconds"`
	HeartbeatIntervalSeconds int      `json:"heartbeatIntervalSeconds"`
	MaxSMSPerHalfHour        int      `json:"maxSmsPerHalfHour"`
	SuccessMatchPatterns     []string `json:"successMatchPatterns"`
	FailureMatchPatterns     []string `json:"failureMatchPatterns"`

	TouchBalanceCheckUssd             string  `json:"touchBalanceCheckUssd"`
	AlfaBalanceCheckUssd              string  `json:"alfaBalanceCheckUssd"`
	TouchThirdPartyRechargeTemplate   string  `json:"touchThirdPartyRechargeTemplate"`
	TouchCreditTransferSmsTemplate    string  `json:"touchCreditTransferSmsTemplate"`
	TouchCreditTransferDestination    string  `json:"touchCreditTransferDestination"`
	AlfaThirdPartyRechargeSmsTemplate string  `json:"alfaThirdPartyRechargeSmsTemplate"`
	AlfaThirdPartyRechargeDestination string  `json:"alfaThirdPartyRechargeDestination"`
	AlfaCreditTransferSmsTemplate     string  `json:"alfaCreditTransferSmsTemplate"`
	AlfaCreditTransferDestination     string  `json:"alfaCreditTransferDestination"`
	TouchMinimumAllowedBalance        float64 `json:"touchMinimumAllowedBalance"`
	AlfaMinimumAllowedBalance         float64 `json:"alfaMinimumAllowedBalance"`
	TouchCreditTransferMessageFee     float64 `json:"touchCreditTransferMessageFee"`
	AlfaCreditTransferMessageFee      float64 `json:"alfaCreditTransferMessageFee"`

	TouchSimBalance      float64 `json:"touchSimBalance"`
	TouchSimValidityDate string  `json:"touchSimValidityDate"`
	AlfaSimBalance       float64 `json:"alfaSimBalance"`
	AlfaSimValidityDate  string  `json:"alfaSimValidityDate"`
}

// commandDTO is the device wire form of a leased command.
type commandDTO struct {
	CommandID       string     `json:"commandId"`
	Provider        string     `json:"provider"`
	CommandType     string     `json:"commandType"`
	Timestamp       int64      `json:"timestamp"`
	RecipientNumber string     `json:"recipientNumber,omitempty"`
	Amount          *float64   `json:"amount,omitempty"`
	CardCode        string     `json:"cardCode,omitempty"`
	Message         string     `json:"message,omitempty"`
	Attempt         int        `json:"attempt"`
	LeaseExpiresAt  *time.Time `json:"leaseExpiresAt,omitempty"`
}

func toCommandDTO(c *Command) commandDTO {
	return commandDTO{
		CommandID:       c.ID.Hex(),
		Provider:        c.Provider,
		CommandType:     c.Type,
		Timestamp:       c.CreatedAt.UnixMilli(),
		RecipientNumber: c.RecipientNumber,
		Amount:          c.Amount,
		CardCode:        c.CardCode,
		Message:         c.Message,
		Attempt:         c.Attempts,
		LeaseExpiresAt:  c.LeaseExpiresAt,
	}
}

// resultDTO is the device's command result report.
type resultDTO struct {
	StatusCode        int      `json:"statusCode"`
	TransferredAmount *float64 `json:"transferredAmount,omitempty"`
	BillingAmount     *float64 `json:"billingAmount,omitempty"`
	Balance           *float64 `json:"balance,omitempty"`
	ValidityDate      string   `json:"validityDate,omitempty"`
	RawReply          string   `json:"rawReply,omitempty"`
	ErrorMessage      string   `json:"errorMessage,omitempty"`
}

// heartbeatDTO is the device's periodic liveness/state report.
type heartbeatDTO struct {
	TouchBalance  *float64 `json:"touchBalance,omitempty"`
	TouchValidity string   `json:"touchValidity,omitempty"`
	AlfaBalance   *float64 `json:"alfaBalance,omitempty"`
	AlfaValidity  string   `json:"alfaValidity,omitempty"`
	AppVersion    string   `json:"appVersion,omitempty"`
}

// logsDTO is a batch of device diagnostic entries.
type logsDTO struct {
	Entries []struct {
		Level     string `json:"level"`
		ErrorCode *int   `json:"errorCode,omitempty"`
		Message   string `json:"message"`
	} `json:"entries"`
}

// Handler serves the device-facing bridge endpoints. Responses are raw JSON (no
// {success,data} envelope) so the device's typed client parses them directly.
type Handler struct {
	svc *Service
}

// NewHandler constructs the device handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// GetConfig serves GET /api/v1/bridge/config.
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	d, ok := DeviceFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "device token required")
		return
	}
	response.JSON(w, http.StatusOK, h.svc.BuildConfig(d))
}

// Poll serves GET /api/v1/bridge/commands?max=N — leases and returns commands.
func (h *Handler) Poll(w http.ResponseWriter, r *http.Request) {
	d, ok := DeviceFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "device token required")
		return
	}
	max := 1
	if v := r.URL.Query().Get("max"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			max = n
		}
	}
	cmds, err := h.svc.Poll(r.Context(), d, max)
	if err != nil {
		response.InternalError(w)
		return
	}
	out := make([]commandDTO, len(cmds))
	for i, c := range cmds {
		out[i] = toCommandDTO(c)
	}
	response.JSON(w, http.StatusOK, out)
}

// PostResult serves POST /api/v1/bridge/commands/{id}/result — idempotent.
func (h *Handler) PostResult(w http.ResponseWriter, r *http.Request) {
	d, ok := DeviceFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "device token required")
		return
	}
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}
	var body resultDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid result body")
		return
	}
	res := &CommandResult{
		StatusCode:        body.StatusCode,
		TransferredAmount: body.TransferredAmount,
		BillingAmount:     body.BillingAmount,
		Balance:           body.Balance,
		ValidityDate:      body.ValidityDate,
		RawReply:          body.RawReply,
		ErrorMessage:      body.ErrorMessage,
	}
	recorded, err := h.svc.IngestResult(r.Context(), id, d.ID, res)
	if err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			response.Error(w, http.StatusConflict, "LEASE_LOST", "this command is no longer leased to your device")
			return
		}
		response.InternalError(w)
		return
	}
	status := "recorded"
	if !recorded {
		status = "duplicate"
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": status})
}

// PostHeartbeat serves POST /api/v1/bridge/heartbeat.
func (h *Handler) PostHeartbeat(w http.ResponseWriter, r *http.Request) {
	d, ok := DeviceFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "device token required")
		return
	}
	var body heartbeatDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid heartbeat body")
		return
	}
	if err := h.svc.Heartbeat(r.Context(), d, HeartbeatInput{
		TouchBalance:  body.TouchBalance,
		TouchValidity: body.TouchValidity,
		AlfaBalance:   body.AlfaBalance,
		AlfaValidity:  body.AlfaValidity,
		AppVersion:    body.AppVersion,
	}); err != nil {
		response.InternalError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostLogs serves POST /api/v1/bridge/logs.
func (h *Handler) PostLogs(w http.ResponseWriter, r *http.Request) {
	d, ok := DeviceFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "device token required")
		return
	}
	var body logsDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid logs body")
		return
	}
	entries := make([]LogEntry, 0, len(body.Entries))
	for _, e := range body.Entries {
		entries = append(entries, LogEntry{Level: e.Level, ErrorCode: e.ErrorCode, Message: e.Message})
	}
	_ = h.svc.StoreLogs(r.Context(), d, entries) // best effort
	w.WriteHeader(http.StatusNoContent)
}

// parsePositiveInt parses a small positive integer (query param).
func parsePositiveInt(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(r-'0')
		if n > 1000 {
			break
		}
	}
	if n <= 0 {
		return 0, errors.New("not positive")
	}
	return n, nil
}
