package checker

import "testing"

func TestClassifyError(t *testing.T) {
	tests := []struct {
		script, status, summary, output, want string
	}{
		{"chk-smtp", "fail", "535 Incorrect authentication data", "", "auth"},
		{"chk-smtp", "fail", "", "Connection unexpectedly closed", "closed"},
		{"chk-smtp", "fail", "", "554 Disabled by user from hPanel", "policy"},
		{"chk-smtp", "fail", "", "550 5.7.1 Sending from domain x.com is not allowed", "policy"},
		{"chk-mysql", "fail", "timeout", "", "timeout"},
		{"chk-mysql", "fail", "host denied", "", "host_denied"},
		{"chk-mysql", "fail", "auth fail", "", "auth"},
		{"chk-postgres", "skip", "SKIP: DB_CONNECTION=mysql", "", "skip"},
		{"chk-redis", "fail", "", "Rede: [Errno 111] Connection refused", "refused"},
		{"chk-smtp", "fail", "", "STARTTLS failed: certificate verify failed", "ssl"},
		{"chk-mysql", "fail", "", "Name or service not known", "dns"},
		{"chk-smtp", "ok", "email sent", "", ""},
	}
	for _, tt := range tests {
		got := ClassifyError(tt.script, tt.status, tt.summary, tt.output)
		if got != tt.want {
			t.Errorf("ClassifyError(%q,%q) = %q, want %q", tt.script, tt.summary, got, tt.want)
		}
	}
}
