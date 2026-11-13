package complaint

import (
	"errors"
	"net/http"

	"github.com/AliSleiman0/salehcard/api/pkg/response"
)

var (
	errNotFound   = errors.New("complaint not found")
	errClosed     = errors.New("complaint is closed")
	errTooMany    = errors.New("too many open complaints")
	errThreadFull = errors.New("complaint thread is full")
	// errConflict is the repository's "guarded update matched nothing" signal
	// (status moved underneath us); the service re-reads and diagnoses it.
	errConflict = errors.New("complaint changed concurrently")
)

// invalidError is a 400 with a customer-presentable message.
type invalidError struct{ msg string }

func (e *invalidError) Error() string { return e.msg }

func invalid(msg string) error { return &invalidError{msg: msg} }

// writeError maps service errors onto HTTP responses. A complaint that is not
// the caller's is a plain 404 (never 403, which would leak that it exists).
func writeError(w http.ResponseWriter, err error) {
	var inv *invalidError
	switch {
	case errors.As(err, &inv):
		response.BadRequest(w, inv.msg)
	case errors.Is(err, errNotFound):
		response.NotFound(w)
	case errors.Is(err, errClosed):
		response.Error(w, http.StatusConflict, "CLOSED", "this complaint is closed")
	case errors.Is(err, errTooMany):
		response.Error(w, http.StatusConflict, "TOO_MANY_OPEN",
			"you already have the maximum number of open complaints; please wait for one to be resolved")
	case errors.Is(err, errThreadFull):
		response.Error(w, http.StatusConflict, "THREAD_FULL", "this complaint has reached its message limit; please open a new one")
	case errors.Is(err, errConflict):
		response.Error(w, http.StatusConflict, "CONFLICT", "the complaint changed, please try again")
	default:
		response.InternalError(w)
	}
}
