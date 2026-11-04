package consumes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KogenAI/kogen-verification/verify"
)

// Signatures kogen-shaping's calls rely on.
var (
	_ *verify.Redactor
	_ func([]string, string) *verify.Redactor = verify.NewRedactor
	_ func(*verify.Redactor, string) string   = (*verify.Redactor).Redact
)

// verify.NewRedactor masks the values of the caller's secret-named environment variables, so
// kogen-shaping never shows or keeps them.
func TestNewRedactorMasksEnvironmentSecrets(t *testing.T) {
	env := []string{"PATH=/usr/bin", "API_TOKEN=tok-9f8e7d6c", "db_password=hunter2", "SECRET_X=tok-9f8e7d6c-and-more", "EMPTY_KEY=", "NAME=plain-value"}
	r := verify.NewRedactor(env, "")
	got := r.Redact("saw tok-9f8e7d6c, hunter2, tok-9f8e7d6c-and-more and plain-value under /usr/bin")
	want := "saw [REDACTED], [REDACTED], [REDACTED] and plain-value under /usr/bin"
	if got != want {
		t.Errorf("verify.NewRedactor: secret-named variables must be masked, longest value first, and others kept:\n got: %s\nwant: %s", got, want)
	}
}

// verify.NewRedactor also masks the string values of Kogen's saved logins under the home it
// is given, only in fields named like a secret.
func TestNewRedactorMasksSavedLogins(t *testing.T) {
	home := t.TempDir()
	creds := filepath.Join(home, ".kogen", "credentials")
	if err := os.MkdirAll(creds, 0o755); err != nil {
		t.Fatal(err)
	}
	login := `{"account":"user@example.invalid","access_token":"at-111222333","tokens":{"refresh_token":"rt-444555666"},"note":"harmless"}`
	if err := os.WriteFile(filepath.Join(creds, "chatgpt.json"), []byte(login), 0o600); err != nil {
		t.Fatal(err)
	}
	got := verify.NewRedactor(nil, home).Redact("at-111222333 rt-444555666 harmless user@example.invalid")
	if want := "[REDACTED] [REDACTED] harmless user@example.invalid"; got != want {
		t.Errorf("verify.NewRedactor: saved login secrets must be masked and other fields kept:\n got: %s\nwant: %s", got, want)
	}
	if got := verify.NewRedactor(nil, "").Redact("at-111222333"); got != "at-111222333" {
		t.Errorf("verify.NewRedactor: without a home no saved login is read, got %q", got)
	}
}

// verify.Redactor.Redact masks labeled secrets without knowing their values; a nil Redactor
// masks only those.
func TestRedactorRedactMasksLabeledSecrets(t *testing.T) {
	cases := map[string]string{
		"Authorization: Bearer abcdef123456 sent": "Authorization: Bearer [REDACTED] sent",
		"got Bearer abcdef123456":                 "got Bearer [REDACTED]",
		"curl --api-key abc123 https://x.invalid": "curl --api-key [REDACTED] https://x.invalid",
		"password=hunter2 user=bob":               "password=[REDACTED] user=bob",
		`{"client_secret": "s3cr3t"}`:             `{"client_secret": [REDACTED]}`,
		"nothing secret here":                     "nothing secret here",
	}
	for in, want := range cases {
		if got := (*verify.Redactor)(nil).Redact(in); got != want {
			t.Errorf("verify.Redactor.Redact: %q must give %q, got %q", in, want, got)
		}
		if got := verify.NewRedactor(nil, "").Redact(in); got != want {
			t.Errorf("verify.Redactor.Redact: %q through a Redactor must give %q, got %q", in, want, got)
		}
	}
	if got := (*verify.Redactor)(nil).Redact("API_TOKEN=tok-9f8e7d6c literal tok-9f8e7d6c"); strings.Contains(got, "API_TOKEN=tok") || !strings.Contains(got, "literal tok-9f8e7d6c") {
		t.Errorf("verify.Redactor.Redact: a nil Redactor masks the labeled assignment only, got %q", got)
	}
	if got := verify.NewRedactor(nil, "").Redact("bad \xff\xfe bytes"); got != "bad � bytes" {
		t.Errorf("verify.Redactor.Redact: bytes that are not UTF-8 must become U+FFFD (a run as one), got %q", got)
	}
}
