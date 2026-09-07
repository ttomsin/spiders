package auth

import "testing"

func TestAuthEncryptDecrypt(t *testing.T) {
	orig := "test_session_token_1234567890abcdef"
	enc, err := encrypt(orig)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if enc == orig {
		t.Fatalf("encrypted matches plain")
	}
	dec, err := decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if dec != orig {
		t.Fatalf("expected %s, got %s", orig, dec)
	}
}

func TestMaskToken(t *testing.T) {
	masked := MaskToken("1234567890abcdef")
	if masked != "1234...cdef" {
		t.Errorf("unexpected mask: %s", masked)
	}
}