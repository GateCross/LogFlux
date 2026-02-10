package svc

import "testing"

func TestGenerateComplexPassword(t *testing.T) {
	password, err := generateComplexPassword(12)
	if err != nil {
		t.Fatalf("generateComplexPassword returned error: %v", err)
	}

	if len(password) != 12 {
		t.Fatalf("expected length 12, got %d", len(password))
	}

	if !validateComplexity(password) {
		t.Fatalf("password does not meet complexity requirements: %s", password)
	}
}

func TestGenerateComplexPasswordMinimumLength(t *testing.T) {
	password, err := generateComplexPassword(4)
	if err != nil {
		t.Fatalf("generateComplexPassword returned error: %v", err)
	}

	if len(password) != 6 {
		t.Fatalf("expected fallback length 6, got %d", len(password))
	}

	if !validateComplexity(password) {
		t.Fatalf("password does not meet complexity requirements: %s", password)
	}
}

func TestGenerateComplexPasswordMaximumLength(t *testing.T) {
	password, err := generateComplexPassword(32)
	if err != nil {
		t.Fatalf("generateComplexPassword returned error: %v", err)
	}

	if len(password) != 18 {
		t.Fatalf("expected fallback length 18, got %d", len(password))
	}

	if !validateComplexity(password) {
		t.Fatalf("password does not meet complexity requirements: %s", password)
	}
}
