package auth

import "testing"

func TestCodeChallengeUsesS256Base64URL(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	challenge := CodeChallengeS256(verifier)

	const want = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if challenge != want {
		t.Fatalf("challenge = %q, want %q", challenge, want)
	}
}

func TestGenerateCodeVerifierIsPKCECompatible(t *testing.T) {
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatal(err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Fatalf("verifier length = %d, want 43..128", len(verifier))
	}
	if verifier != stringsTrimPKCEUnsafe(verifier) {
		t.Fatalf("verifier contains PKCE-unsafe characters: %q", verifier)
	}
}

func stringsTrimPKCEUnsafe(value string) string {
	out := make([]rune, 0, len(value))
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == '_' || r == '~' {
			out = append(out, r)
		}
	}
	return string(out)
}
