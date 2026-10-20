package password

import (
	"strings"
	"testing"
)

func TestGenerateMatchesValidate(t *testing.T) {
	for _, length := range []int{0, 8, 12, 16, 72, 100} {
		got, err := Generate(length)
		if err != nil {
			t.Fatalf("Generate(%d) error: %v", length, err)
		}
		if err := Validate(got); err != nil {
			t.Fatalf("Generate(%d) produced invalid password %q: %v", length, got, err)
		}
		expected := length
		if expected < MinLength {
			expected = MinLength
		}
		if expected > MaxLength {
			expected = MaxLength
		}
		if len(got) != expected {
			t.Fatalf("Generate(%d) length = %d, want %d", length, len(got), expected)
		}
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name  string
		input string
		ok    bool
	}{
		{"letters and digits", "abcdef12", true},
		{"too short", "abc123", false},
		{"letters only", "abcdefgh", false},
		{"digits only", "12345678", false},
		{"too long", "a1" + string(make([]byte, MaxLength)), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.input)
			if tc.ok && err != nil {
				t.Fatalf("Validate(%q) = %v, want ok", tc.input, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("Validate(%q) = nil, want error", tc.input)
			}
		})
	}
}

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("secret123")
	if err != nil {
		t.Fatalf("Hash error: %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("Hash result is not a bcrypt hash: %q", hash)
	}
	if err := Verify(hash, "secret123"); err != nil {
		t.Fatalf("Verify correct password error: %v", err)
	}
	if err := Verify(hash, "wrong123"); err == nil {
		t.Fatal("Verify wrong password = nil, want error")
	}
}
