package auth

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	originalToken := "sample_secret_twitter_auth_token_1234567890abcdef"
	enc, err := encrypt(originalToken)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if enc == originalToken {
		t.Fatalf("encrypted token is identical to plaintext")
	}

	dec, err := decrypt(enc)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if dec != originalToken {
		t.Fatalf("expected decrypted %s, got %s", originalToken, dec)
	}
}

func TestMaskToken(t *testing.T) {
	token := "abcdef1234567890"
	masked := MaskToken(token)
	expected := "abcd...7890"
	if masked != expected {
		t.Fatalf("expected %s, got %s", expected, masked)
	}
}

func TestMultiTokenStorage(t *testing.T) {
	tokens := []string{
		"token_account_one_1234567890abcdef",
		"token_account_two_0987654321fedcba",
	}

	var encs []string
	for _, tok := range tokens {
		enc, err := encrypt(tok)
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}
		encs = append(encs, enc)
	}

	for i, enc := range encs {
		dec, err := decrypt(enc)
		if err != nil {
			t.Fatalf("decryption failed: %v", err)
		}
		if dec != tokens[i] {
			t.Fatalf("expected %s, got %s", tokens[i], dec)
		}
	}
}

