package order

import (
	"testing"

	"github.com/AliSleiman0/salehcard/api/internal/modules/product"
)

func TestValidateFieldInputs_FilePhoto(t *testing.T) {
	p := &product.Product{InputFields: []product.InputField{
		{Key: "qr", Label: product.I18nLabel{En: "QR code"}, Type: product.InputFieldFile},
	}}
	cases := []struct {
		name  string
		value string
		ok    bool
	}{
		{"our upload", "https://acct.blob.core.windows.net/c/order-files/6af0.jpg", true},
		{"local dev upload", "http://localhost:8090/uploads/order-files/abc.jpg", true},
		{"empty (not enforced server-side)", "", true},
		{"foreign link", "https://evil.example/qr.jpg", false},
		{"other keyspace", "https://x/uploads/kyc/abc.jpg", false},
		{"plain text", "123456", false},
	}
	for _, c := range cases {
		err := validateFieldInputs(p, []OrderFieldInput{{Key: "qr", Value: c.value}})
		if (err == nil) != c.ok {
			t.Errorf("%s: ok=%v, err=%v", c.name, c.ok, err)
		}
	}
}
