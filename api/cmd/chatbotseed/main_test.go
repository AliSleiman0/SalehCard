package main

import "testing"

func TestValidWhatsapp(t *testing.T) {
	for _, ok := range []string{"+961 71 363 772", "96171363772", "(961) 71-363-772"} {
		if !validWhatsapp(ok) {
			t.Errorf("%q should be accepted", ok)
		}
	}
	for _, bad := range []string{"", "12345", "+961 71 abc", "71+363772", "https://wa.me/96171363772", "1234567890123456"} {
		if validWhatsapp(bad) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
