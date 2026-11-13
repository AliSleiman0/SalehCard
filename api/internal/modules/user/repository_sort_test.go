package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AliSleiman0/salehcard/api/pkg/pagination"
)

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
