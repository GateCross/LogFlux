package svc

import "testing"

func TestGenerateComplexPassword(t *testing.T) {
	password, err := generateComplexPassword(20)
	if err != nil {
		t.Fatalf("generateComplexPassword returned error: %v", err)
	}

	if len(password) != 20 {
		t.Fatalf("expected length 20, got %d", len(password))
	}

	if !validateComplexity(password) {
		t.Fatalf("password does not meet complexity requirements: %s", password)
	}
}

func TestGenerateComplexPasswordMinimumLength(t *testing.T) {
	password, err := generateComplexPassword(6)
	if err != nil {
		t.Fatalf("generateComplexPassword returned error: %v", err)
	}

	if len(password) != 12 {
		t.Fatalf("expected fallback length 12, got %d", len(password))
	}

	if !validateComplexity(password) {
		t.Fatalf("password does not meet complexity requirements: %s", password)
	}
}

