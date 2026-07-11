package kyc

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/internal/platform/blob"
	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
)

// Purger permanently removes a user's KYC footprint: the stored document
// photos (blobs) first, then the submission row. Account deletion runs it
// BEFORE flipping the account to deleted, so ID photos can never outlive an
// account that reads as deleted.
type Purger struct {
	repo  Repository
	store blob.Storage
}

// NewPurger constructs a Purger over the submission repository and the shared
// blob store the document photos were uploaded to.
func NewPurger(repo Repository, store blob.Storage) *Purger {
	return &Purger{repo: repo, store: store}
}

// PurgeByUser deletes the user's document photos and submission row, reporting
// whether a submission existed. A blob-delete failure ABORTS before the row
// delete: the caller retries later, and a dangling row still points at the
// photos — the reverse order would orphan ID photos nothing references. No
// submission is a clean no-op. Blob deletes are idempotent (see blob.Storage),
// so a retry after a partial purge is safe.
func (p *Purger) PurgeByUser(ctx context.Context, userID bson.ObjectID) (bool, error) {
	sub, err := p.repo.FindByUserID(ctx, userID)
	if err != nil {
		if err == apperrors.ErrNotFound {
			return false, nil
		}
		return false, err
	}

	// DocumentBackURL is empty for passports; keyFromDocURL maps "" to "".
	for _, docURL := range []string{sub.DocumentFrontURL, sub.DocumentBackURL} {
		key := keyFromDocURL(docURL)
		if key == "" {
			if docURL != "" {
				// Off-keyspace URL (validDocURL should make this impossible):
				// there is no blob of ours behind it, so skip rather than abort.
				// The URL itself is never logged — it could embed anything.
				slog.Warn("kyc: purge skipping a document URL outside the kyc keyspace", "userId", userID.Hex())
			}
			continue
		}
		if err := p.store.Delete(ctx, key); err != nil {
			return false, fmt.Errorf("kyc: purge document blob: %w", err)
		}
	}

	if _, err := p.repo.DeleteByUserID(ctx, userID); err != nil {
		if err == apperrors.ErrNotFound {
			return true, nil // raced a concurrent purge — the row is gone either way
		}
		return false, err
	}
	return true, nil
}

// keyFromDocURL derives the blob-storage key from a stored document URL. Both
// adapters keep the kyc/ keyspace visible in the URL path (azure:
// /<container>/kyc/<hex>.jpg, local: /uploads/kyc/<hex>.jpg), so the key is
// the path from its "/kyc/" segment on. Junk and off-keyspace URLs map to ""
// (the caller decides whether that warrants a warning).
func keyFromDocURL(docURL string) string {
	if docURL == "" {
		return ""
	}
	u, err := url.Parse(docURL)
	if err != nil {
		return ""
	}
	idx := strings.Index(u.Path, "/kyc/")
	if idx < 0 {
		return ""
	}
	return u.Path[idx+1:]
}
