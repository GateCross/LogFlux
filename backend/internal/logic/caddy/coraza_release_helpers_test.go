package caddy

import "testing"

func TestExtractCorazaVersionFromText(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "module path with version",
			input:  "http.handlers.coraza_waf github.com/corazawaf/coraza-caddy/v2@v2.1.0",
			output: "v2.1.0",
		},
		{
			name:   "coraza line fallback semver",
			input:  "module=coraza-waf version=v2.1.1-beta.1",
			output: "v2.1.1-beta.1",
		},
		{
			name:   "no coraza",
			input:  "http.handlers.reverse_proxy github.com/caddyserver/caddy/v2@v2.9.0",
			output: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := extractCorazaVersionFromText(testCase.input)
			if got != testCase.output {
				t.Fatalf("unexpected version, want=%q got=%q", testCase.output, got)
			}
		})
	}
}

func TestIsCommandNotFoundError(t *testing.T) {
	notFoundErr := "exec caddy list-modules --versions failed: fork/exec caddy: executable file not found in $PATH"
	if !isCommandNotFoundError(assertErr(notFoundErr)) {
		t.Fatalf("expected not found error to be recognized")
	}

	otherErr := "exec caddy list-modules --versions failed: signal: killed"
	if isCommandNotFoundError(assertErr(otherErr)) {
		t.Fatalf("expected non not-found error")
	}
}

type staticErr string

func (e staticErr) Error() string {
	return string(e)
}

func assertErr(messageText string) error {
	return staticErr(messageText)
}
