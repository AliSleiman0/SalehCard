package product

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNaturalLess(t *testing.T) {
	assert.True(t, naturalLess("apple", "Bulala"), "case-insensitive")
	assert.True(t, naturalLess("JAWAKER 150", "JAWAKER 400"))
	assert.True(t, naturalLess("JAWAKER 400", "jawaker 1000"), "numeric, not byte-wise")
	assert.False(t, naturalLess("JAWAKER 1,000", "JAWAKER 1000"), "commas inside numbers ignored")
	assert.False(t, naturalLess("JAWAKER 1000", "JAWAKER 1,000"))
	assert.True(t, naturalLess("2$ ALFA", "10$ ALFA"))
	assert.True(t, naturalLess("abc", "abcd"))
}

func TestSortProductsByName(t *testing.T) {
	ps := []Product{
		{Title: I18nString{En: "Yalla Ludo"}},
		{Title: I18nString{En: "", Ar: "باقة"}},
		{Title: I18nString{En: "bigo live 1000"}},
		{Title: I18nString{En: "Bigo Live 200"}},
		{Title: I18nString{}},
		{Title: I18nString{En: "(Sigma Plus 1 Year"}},
	}
	sortProductsByName(ps)
	got := []string{}
	for i := range ps {
		got = append(got, sortKey(&ps[i]))
	}
	assert.Equal(t, []string{"Bigo Live 200", "bigo live 1000", "Sigma Plus 1 Year", "Yalla Ludo", "باقة", ""}, got)
}
