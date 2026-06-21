package checker

import "testing"

func TestSummarizeOutputPrefersSummary(t *testing.T) {
	out := "line1\nSUMMARY: 3 DBs ~1.2 GB\nOK"
	got := SummarizeOutput(out)
	if got != "3 DBs ~1.2 GB" {
		t.Fatalf("got %q", got)
	}
}

func TestClassifyStatusSkip(t *testing.T) {
	if ClassifyStatus(2, "SKIP: foo") != "skip" {
		t.Fatal("expected skip")
	}
}

func TestClassifyStatusSMTPSuccess(t *testing.T) {
	out := "Batch SMTP → rootmasters@proton.me\nSUMMARY: email → rootmasters@proton.me\nOK — email enviado → rootmasters@proton.me"
	if ClassifyStatus(0, out) != "ok" {
		t.Fatal("expected ok for smtp success output")
	}
}

func TestClassifyStatusSuccessMarkerDespiteFalhaSubstring(t *testing.T) {
	out := "log line about falha anterior\nOK — email enviado → rootmasters@proton.me"
	if ClassifyStatus(0, out) != "ok" {
		t.Fatal("success marker should win over falha substring")
	}
}

func TestClassifyStatusConnectFailSummary(t *testing.T) {
	out := "Falha: timeout\nSUMMARY: connect fail"
	if ClassifyStatus(1, out) != "fail" {
		t.Fatal("SUMMARY: connect fail must not classify as ok")
	}
}

func TestClassifyStatusHostDeniedSummary(t *testing.T) {
	out := "Falha: 1130\nSUMMARY: host denied"
	if ClassifyStatus(1, out) != "fail" {
		t.Fatal("SUMMARY: host denied must classify as fail")
	}
}

func TestClassifyStatusExplicitFailure(t *testing.T) {
	if ClassifyStatus(1, "Falha SMTP: auth failed") != "fail" {
		t.Fatal("expected fail")
	}
}
