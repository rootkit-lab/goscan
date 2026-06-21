package checker

import (
	"regexp"
	"strings"
)

var (
	timeoutRe    = regexp.MustCompile(`(?i)(timed?\s*out|timeout|excedeu\s+\d+s|time.?out|timeout expired)`)
	authRe       = regexp.MustCompile(`(?i)(535|authentication|auth(entication)?\s+(failed|fail|error)|incorrect.*(user|pass|auth)|invalid credentials|access denied|1045|28p01|password authentication failed|erro de autentica)`)
	dnsRe        = regexp.MustCompile(`(?i)(name or service not known|nodename nor servname|getaddrinfo|unknown host|no address associated|dns fail|errno -2)`)
	sslRe        = regexp.MustCompile(`(?i)(ssl|tls|certificate|cert_verify|starttls|wrong version number|handshake)`)
	refusedRe    = regexp.MustCompile(`(?i)(connection refused|actively refused|ECONNREFUSED|errno 111|connect refused)`)
	compatRe     = regexp.MustCompile(`(?i)(skip:|incompat|wrong driver|use chk-)`)
	closedRe     = regexp.MustCompile(`(?i)(connection unexpectedly closed|broken pipe|reset by peer)`)
	policyRe     = regexp.MustCompile(`(?i)(550|554|5\.7\.1|sending from domain|disabled by user|not allowed.*domain|policy)`)
	hostDeniedRe = regexp.MustCompile(`(?i)(1130|host denied|host .* is not allowed|not allowed to connect to this)`)
)

// ClassifyError maps checker output to a coarse error bucket for batch analysis.
func ClassifyError(scriptID, status, summary, output string) string {
	if status == "ok" {
		return ""
	}
	if status == "skip" {
		return "skip"
	}

	text := strings.ToLower(StripANSI(summary + "\n" + output))

	switch {
	case compatRe.MatchString(text):
		return "skip"
	case policyRe.MatchString(text):
		return "policy"
	case hostDeniedRe.MatchString(text):
		return "host_denied"
	case authRe.MatchString(text):
		return "auth"
	case dnsRe.MatchString(text):
		return "dns"
	case timeoutRe.MatchString(text):
		return "timeout"
	case sslRe.MatchString(text):
		return "ssl"
	case refusedRe.MatchString(text):
		return "refused"
	case closedRe.MatchString(text):
		return "closed"
	}

	if strings.Contains(scriptID, "smtp") || strings.Contains(scriptID, "mail") {
		return "smtp_other"
	}
	if isDBScript(scriptID) {
		return "db_other"
	}
	return "other"
}

func isDBScript(id string) bool {
	switch id {
	case "chk-mysql", "chk-postgres", "chk-redis", "chk-mongodb", "chk-memcached":
		return true
	default:
		return false
	}
}

func IsSMTPScript(id string) bool {
	switch id {
	case "chk-smtp", "chk-sendgrid", "chk-mailgun", "chk-twilio":
		return true
	default:
		return false
	}
}

func IsDBScript(id string) bool {
	return isDBScript(id)
}

func ScriptEngine(id string) string {
	switch id {
	case "chk-mysql":
		return "MySQL"
	case "chk-postgres":
		return "PostgreSQL"
	case "chk-redis":
		return "Redis"
	case "chk-mongodb":
		return "MongoDB"
	case "chk-memcached":
		return "Memcached"
	default:
		return ""
	}
}
