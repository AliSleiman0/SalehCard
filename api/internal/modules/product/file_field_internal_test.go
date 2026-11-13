package product

import "testing"

func TestSanitizeInputFields_FileType(t *testing.T) {
	min := 1.0
	got, err := sanitizeInputFields([]InputField{{
		Key:         " qr ",
		Type:        InputFieldFile,
		Constraints: &InputFieldConstraints{Min: &min, Options: []string{"x"}},
	}})
	if err != nil {
		t.Fatalf("file type must be accepted: %v", err)
	}
	if got[0].Key != "qr" || got[0].Type != InputFieldFile || got[0].Constraints != nil {
		t.Fatalf("file field not normalized: %+v", got[0])
	}
}
