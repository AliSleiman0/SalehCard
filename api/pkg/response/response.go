package response

import (
	"encoding/json"
	"net/http"
)

// ErrorDetail holds a machine-readable code and a human-readable message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Response is the standard envelope returned by every API endpoint.
type Response struct {
	Success bool         `json:"success"`
	Data    any          `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
	Meta    any          `json:"meta,omitempty"`
}

// JSON encodes v as JSON and writes it to w with the given HTTP status code.
// It always sets Content-Type: application/json.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Error writes a failed Response envelope with the supplied status, code, and message.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// OK writes a successful 200 Response with data.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// OKWithMeta writes a successful 200 Response with both data and meta fields.
func OKWithMeta(w http.ResponseWriter, data any, meta any) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// NotFound writes a 404 Response.
func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
}

// BadRequest writes a 400 Response with the supplied message.
func BadRequest(w http.ResponseWriter, msg string) {
	Error(w, http.StatusBadRequest, "BAD_REQUEST", msg)
}

// InternalError writes a 500 Response with a generic message.
func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}

// Forbidden writes a 403 Response.
func Forbidden(w http.ResponseWriter, msg string) {
	Error(w, http.StatusForbidden, "FORBIDDEN", msg)
}

// Unauthorized writes a 401 Response.
func Unauthorized(w http.ResponseWriter, msg string) {
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED", msg)
}

// NotImplemented writes a 501 Response for stubbed admin endpoints.
func NotImplemented(w http.ResponseWriter, feature string) {
	Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", feature+" is not implemented yet")
}

// Stub returns an http.HandlerFunc that always responds 501 NotImplemented.
// Used to register the admin route map for modules whose handlers are TODO.
func Stub(feature string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		NotImplemented(w, feature)
	}
}
