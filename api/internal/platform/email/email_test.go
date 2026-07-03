package email

import "testing"

func TestNew_LogDefault(t *testing.T) {
	s, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(LogSender); !ok {
		t.Fatalf("empty provider should be LogSender, got %T", s)
	}
}

func TestNew_UnknownProviderErrors(t *testing.T) {
	if _, err := New(Config{Provider: "carrierpigeon"}); err == nil {
		t.Fatal("unknown provider should error")
	}
}

func TestNew_SMTPRequiresHostAndFrom(t *testing.T) {
	if _, err := New(Config{Provider: "smtp"}); err == nil {
		t.Fatal("smtp without host/from should error")
	}
}

func TestNew_SendGridRequiresKeyAndFrom(t *testing.T) {
	if _, err := New(Config{Provider: "sendgrid"}); err == nil {
		t.Fatal("sendgrid without key/from should error")
	}
}
