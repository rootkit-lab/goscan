package checker

import (
	"regexp"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func SummarizeOutput(s string) string {
	s = StripANSI(s)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SUMMARY:") {
			msg := strings.TrimSpace(strings.TrimPrefix(line, "SUMMARY:"))
			if len(msg) > 350 {
				return msg[:350] + "…"
			}
			return msg
		}
	}
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "↳") || strings.HasPrefix(line, "[") {
			continue
		}
		if len(line) > 200 {
			return line[:200] + "…"
		}
		return line
	}
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

func summaryMsgIsFailure(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))
	for _, kw := range []string{"fail", "falha", "erro", "timeout", "denied", "refused"} {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

func summaryIndicatesFailure(lower string) bool {
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "summary:") {
			continue
		}
		if summaryMsgIsFailure(strings.TrimPrefix(line, "summary:")) {
			return true
		}
	}
	return false
}

func outputHasSuccessMarker(lower string) bool {
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "summary:") {
			msg := strings.TrimPrefix(line, "summary:")
			if summaryMsgIsFailure(msg) {
				continue
			}
			return true
		}
		if strings.HasPrefix(line, "ok —") || strings.HasPrefix(line, "ok -") {
			return true
		}
		if strings.HasPrefix(line, "ok:") {
			return true
		}
	}
	return false
}

func outputHasExplicitFailure(lower string) bool {
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "skip:") {
			continue
		}
		if strings.HasPrefix(line, "falha") || strings.HasPrefix(line, "falha:") {
			return true
		}
		if strings.HasPrefix(line, "falha smtp:") {
			return true
		}
		if strings.HasPrefix(line, "erro http") || strings.HasPrefix(line, "erro ") {
			return true
		}
		if strings.HasPrefix(line, "rede:") {
			return true
		}
		if strings.HasPrefix(line, "chave inválida") || strings.HasPrefix(line, "token inválido") {
			return true
		}
	}
	return false
}

func ClassifyStatus(exitCode int, output string) string {
	lower := strings.ToLower(StripANSI(output))
	if strings.Contains(lower, "skip:") || exitCode == 2 {
		return "skip"
	}
	if summaryIndicatesFailure(lower) || outputHasExplicitFailure(lower) {
		return "fail"
	}
	if outputHasSuccessMarker(lower) {
		return "ok"
	}
	if exitCode == 0 {
		return "ok"
	}
	if exitCode < 0 {
		return "fail"
	}
	return "fail"
}
