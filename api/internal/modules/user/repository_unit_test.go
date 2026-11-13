package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

// keyed flattens a bson.D into a map for order-independent assertions.
func keyed(d bson.D) map[string]any {
	m := make(map[string]any, len(d))
	for _, e := range d {
		m[e.Key] = e.Value
	}
	return m
}

func TestUserFilter_Build_Empty(t *testing.T) {
	// An unfiltered listing still hides soft-deleted accounts by default.
	m := keyed(UserFilter{}.build())
	assert.Equal(t, bson.D{{Key: "$ne", Value: StatusDeleted}}, m["status"])
}

func TestUserFilter_Build_DeletedIsExactMatch(t *testing.T) {
	// Explicitly filtering status=deleted surfaces the soft-deleted accounts.
	m := keyed(UserFilter{Status: StatusDeleted}.build())
	assert.Equal(t, StatusDeleted, m["status"])
}

func TestUserFilter_Build_RoleAndStatus(t *testing.T) {
	m := keyed(UserFilter{Role: RoleReseller, Status: StatusSuspended}.build())
	assert.Equal(t, RoleReseller, m["role"])
	assert.Equal(t, StatusSuspended, m["status"])
}

func TestUserFilter_Build_ActiveMatchesMissingStatus(t *testing.T) {
	// "active" must also catch legacy accounts with no stored status, so it
	// becomes an $or over {status:active} and {status:{$exists:false}}.
	m := keyed(UserFilter{Status: StatusActive}.build())
	or, ok := m["$or"].(bson.A)
	require.True(t, ok, "active status should produce an $or")
	assert.Len(t, or, 2)
}

func TestUserFilter_Build_ResellerTier(t *testing.T) {
	// The reseller listing forces role=reseller and filters by tier name.
	m := keyed(UserFilter{Role: RoleReseller, ResellerTier: "Gold"}.build())
	assert.Equal(t, RoleReseller, m["role"])
	assert.Equal(t, "Gold", m["resellerTier"])
}

func TestUserFilter_Build_SearchHexMatchesID(t *testing.T) {
	id := bson.NewObjectID()
	m := keyed(UserFilter{Search: id.Hex()}.build())
	assert.Equal(t, id, m["_id"])
	assert.NotContains(t, m, "$or")
}

func TestUserFilter_Build_SearchTextMatchesEmailOrPhone(t *testing.T) {
	m := keyed(UserFilter{Search: "sara"}.build())
	or, ok := m["$or"].(bson.A)
	require.True(t, ok, "non-hex search should produce an $or over email/phone")
	assert.Len(t, or, 2)
	assert.NotContains(t, m, "_id")
}

func TestUserFilterSortDoc(t *testing.T) {
	assert.Equal(t, bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}, UserFilter{}.sortDoc())
	assert.Equal(t, bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}, UserFilter{Sort: "bogus"}.sortDoc())
	assert.Equal(t, bson.D{{Key: "name", Value: 1}, {Key: "email", Value: 1}, {Key: "_id", Value: 1}}, UserFilter{Sort: "name"}.sortDoc())
	assert.Equal(t, bson.D{{Key: "walletBalance", Value: -1}, {Key: "_id", Value: -1}}, UserFilter{Sort: "-wallet"}.sortDoc())
}

func TestUserFilterNameSortPipeline(t *testing.T) {
	_, ok := UserFilter{Sort: "wallet"}.nameSortPipeline(bson.D{}, pagination.Params{Page: 1, Limit: 20})
	assert.False(t, ok, "only the name sort aggregates")

	pl, ok := UserFilter{Sort: "-name"}.nameSortPipeline(bson.D{{Key: "role", Value: "customer"}}, pagination.Params{Page: 2, Limit: 20})
	require.True(t, ok)
	require.Len(t, pl, 5)
	assert.Equal(t, "$match", pl[0][0].Key)
	assert.Equal(t, bson.D{{Key: "_sortName", Value: -1}, {Key: "_id", Value: -1}}, pl[2][0].Value)
	assert.EqualValues(t, 20, pl[3][0].Value) // page 2 skips the first 20
}
