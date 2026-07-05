package kyc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	apperrors "github.com/AliSleiman0/salehcard/api/pkg/errors"
	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// fakeRepo is an in-memory Repository keyed by userId (one submission per user).
type fakeRepo struct {
	byUser map[bson.ObjectID]*Submission
}

func newFakeRepo() *fakeRepo { return &fakeRepo{byUser: map[bson.ObjectID]*Submission{}} }

func (f *fakeRepo) FindByUserID(_ context.Context, userID bson.ObjectID) (*Submission, error) {
	if s, ok := f.byUser[userID]; ok {
		return s, nil
	}
	return nil, apperrors.ErrNotFound
}

// Upsert mirrors MongoRepository.Upsert's explicit per-field $set/$unset list
// (repository.go) rather than storing the caller's struct wholesale — a field
// missing from the real list must also be missing here, so a Submission field
// that is validated but never persisted fails the persistence tests. KEEP IN
// SYNC with the real Upsert's field list.
func (f *fakeRepo) Upsert(_ context.Context, s *Submission) (*Submission, error) {
	stored := &Submission{
		ID:               bson.NewObjectID(),
		UserID:           s.UserID,
		FullName:         s.FullName,
		DateOfBirth:      s.DateOfBirth,
		PlaceOfBirth:     s.PlaceOfBirth,
		PlaceOfResidence: s.PlaceOfResidence,
		DocumentType:     s.DocumentType,
		DocumentNumber:   s.DocumentNumber,
		DocumentFrontURL: s.DocumentFrontURL,
		Status:           StatusPending,
	}
	if s.DocumentBackURL != "" {
		stored.DocumentBackURL = s.DocumentBackURL
	}
	f.byUser[s.UserID] = stored
	return stored, nil
}

func (f *fakeRepo) FindByID(context.Context, bson.ObjectID) (*Submission, error) {
	return nil, apperrors.ErrNotFound
}

func (f *fakeRepo) List(context.Context, Filter, pagination.Params) ([]Row, int64, error) {
	return nil, 0, nil
}

func (f *fakeRepo) UpdateStatus(context.Context, bson.ObjectID, string, string, string) (*Submission, error) {
	return nil, nil
}

func (f *fakeRepo) CountPending(context.Context) (int64, error) { return 0, nil }

func validInput() SubmitInput {
	return SubmitInput{
		FullName:         "Jane Doe",
		DateOfBirth:      "1995-04-12",
		PlaceOfBirth:     "Beirut",
		PlaceOfResidence: "Beirut",
		DocumentType:     DocPassport,
		DocumentNumber:   "RL1234567",
		DocumentFrontURL: "https://cdn.test/uploads/kyc/abc.jpg",
	}
}

func TestSubmit_Valid_GoesPending(t *testing.T) {
	svc := NewService(newFakeRepo())
	p, err := svc.Submit(context.Background(), bson.NewObjectID(), validInput())
	require.NoError(t, err)
	assert.Equal(t, ProfilePending, p.Status)
	require.NotNil(t, p.Submission)
	assert.Equal(t, "Jane Doe", p.Submission.FullName)
}

func TestSubmit_Validation(t *testing.T) {
	svc := NewService(newFakeRepo())
	cases := map[string]func(SubmitInput) SubmitInput{
		"empty name":    func(in SubmitInput) SubmitInput { in.FullName = "  "; return in },
		"bad dob":       func(in SubmitInput) SubmitInput { in.DateOfBirth = "12/04/1995"; return in },
		"empty pob":     func(in SubmitInput) SubmitInput { in.PlaceOfBirth = ""; return in },
		"empty res":     func(in SubmitInput) SubmitInput { in.PlaceOfResidence = ""; return in },
		"bad doc type":  func(in SubmitInput) SubmitInput { in.DocumentType = "ssn"; return in },
		"empty doc num": func(in SubmitInput) SubmitInput { in.DocumentNumber = ""; return in },
		"missing front photo": func(in SubmitInput) SubmitInput { in.DocumentFrontURL = "  "; return in },
		"front not a URL":     func(in SubmitInput) SubmitInput { in.DocumentFrontURL = "not-a-url"; return in },
		"front off keyspace":  func(in SubmitInput) SubmitInput { in.DocumentFrontURL = "https://cdn.test/x.jpg"; return in },
		"front bad scheme":    func(in SubmitInput) SubmitInput { in.DocumentFrontURL = "ftp://cdn.test/kyc/a.jpg"; return in },
		"id_card missing back": func(in SubmitInput) SubmitInput {
			in.DocumentType = DocIDCard
			return in
		},
		"license missing back": func(in SubmitInput) SubmitInput {
			in.DocumentType = DocLicense
			return in
		},
		"junk back photo": func(in SubmitInput) SubmitInput {
			in.DocumentType = DocIDCard
			in.DocumentBackURL = "not-a-url"
			return in
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Submit(context.Background(), bson.NewObjectID(), mutate(validInput()))
			require.Error(t, err)
			assert.ErrorIs(t, err, apperrors.ErrBadRequest)
		})
	}
}

// TestSubmit_PassportBackOptional: passports are single-sided, so the back
// photo may be omitted (validInput uses DocPassport with no back URL).
func TestSubmit_PassportBackOptional(t *testing.T) {
	svc := NewService(newFakeRepo())
	p, err := svc.Submit(context.Background(), bson.NewObjectID(), validInput())
	require.NoError(t, err)
	assert.Equal(t, ProfilePending, p.Status)
	assert.Empty(t, p.Submission.DocumentBackURL)
}

func TestSubmit_IDCardWithBothPhotos_OK(t *testing.T) {
	svc := NewService(newFakeRepo())
	in := validInput()
	in.DocumentType = DocIDCard
	in.DocumentBackURL = "https://cdn.test/uploads/kyc/def.jpg"
	p, err := svc.Submit(context.Background(), bson.NewObjectID(), in)
	require.NoError(t, err)
	assert.Equal(t, ProfilePending, p.Status)
	assert.Equal(t, "https://cdn.test/uploads/kyc/abc.jpg", p.Submission.DocumentFrontURL)
	assert.Equal(t, "https://cdn.test/uploads/kyc/def.jpg", p.Submission.DocumentBackURL)
}

// TestSubmit_PersistsDocumentURLs: the photo URLs must survive the repository
// round trip (regression: Upsert's $set list originally omitted them, so they
// validated fine and then silently dropped).
func TestSubmit_PersistsDocumentURLs(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	uid := bson.NewObjectID()
	in := validInput()
	in.DocumentType = DocIDCard
	in.DocumentBackURL = "https://cdn.test/uploads/kyc/def.jpg"
	_, err := svc.Submit(context.Background(), uid, in)
	require.NoError(t, err)

	p, err := svc.GetProfile(context.Background(), uid)
	require.NoError(t, err)
	require.NotNil(t, p.Submission)
	assert.Equal(t, "https://cdn.test/uploads/kyc/abc.jpg", p.Submission.DocumentFrontURL)
	assert.Equal(t, "https://cdn.test/uploads/kyc/def.jpg", p.Submission.DocumentBackURL)
}

// TestSubmit_PassportResubmitClearsBackURL: resubmitting as a passport (no back
// photo) must clear a stale back URL left by a prior id_card submission.
func TestSubmit_PassportResubmitClearsBackURL(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	uid := bson.NewObjectID()

	first := validInput()
	first.DocumentType = DocIDCard
	first.DocumentBackURL = "https://cdn.test/uploads/kyc/def.jpg"
	_, err := svc.Submit(context.Background(), uid, first)
	require.NoError(t, err)

	_, err = svc.Submit(context.Background(), uid, validInput()) // passport, no back
	require.NoError(t, err)

	p, err := svc.GetProfile(context.Background(), uid)
	require.NoError(t, err)
	require.NotNil(t, p.Submission)
	assert.Equal(t, "https://cdn.test/uploads/kyc/abc.jpg", p.Submission.DocumentFrontURL)
	assert.Empty(t, p.Submission.DocumentBackURL)
}

func TestGetProfile_NoSubmission_Unverified(t *testing.T) {
	svc := NewService(newFakeRepo())
	p, err := svc.GetProfile(context.Background(), bson.NewObjectID())
	require.NoError(t, err)
	assert.Equal(t, ProfileUnverified, p.Status)
	assert.Nil(t, p.Submission)
}

func TestGetProfile_ApprovedReadsVerified(t *testing.T) {
	repo := newFakeRepo()
	uid := bson.NewObjectID()
	repo.byUser[uid] = &Submission{UserID: uid, Status: StatusApproved}
	svc := NewService(repo)
	p, err := svc.GetProfile(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, ProfileVerified, p.Status)
}

func TestGetProfile_RejectedCarriesReason(t *testing.T) {
	repo := newFakeRepo()
	uid := bson.NewObjectID()
	repo.byUser[uid] = &Submission{UserID: uid, Status: StatusRejected, RejectionReason: "blurry name"}
	svc := NewService(repo)
	p, err := svc.GetProfile(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, ProfileRejected, p.Status)
	assert.Equal(t, "blurry name", p.RejectionReason)
}
