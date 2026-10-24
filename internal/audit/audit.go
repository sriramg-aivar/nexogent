// Package audit records an append-only activity log and exports it (F11). Pure.
package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"time"
)

// Action constants.
const (
	LoginSuccess   = "login.success"
	LoginFailure   = "login.failure"
	ScanFull       = "scan.full"
	ScanQuick      = "scan.quick"
	ScanDeepStart  = "scan.deep.start"
	ScanDeepFinish = "scan.deep.finish"
	RiskAccept     = "risk.accept"
	RiskRevoke     = "risk.revoke"
	RiskExpire     = "risk.expire"
	PlanUpdate     = "plan.update"
	SettingsUpdate = "settings.update"
	ReportCreate   = "report.create"
	ReportSubmit   = "report.submit"
	ReportApprove  = "report.approve"
	ReportReject   = "report.reject"
	ReportDownload = "report.download"
	AlertSent      = "alert.sent"
	AlertFailed    = "alert.failed"
)

// Entry is one audit-log record.
type Entry struct {
	At         time.Time `json:"at"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	Target     string    `json:"target"`
	Detail     string    `json:"detail"`
	IP         string    `json:"ip"`
}

// ExportCSV renders entries as CSV with formula-injection escaping.
func ExportCSV(entries []Entry) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"at", "actor", "action", "target_type", "target", "detail", "ip"})
	for _, e := range entries {
		_ = w.Write([]string{
			e.At.UTC().Format(time.RFC3339),
			csvSafe(e.Actor), csvSafe(e.Action), csvSafe(e.TargetType),
			csvSafe(e.Target), csvSafe(e.Detail), csvSafe(e.IP),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// ExportJSON renders entries as pretty JSON.
func ExportJSON(entries []Entry) ([]byte, error) {
	return json.MarshalIndent(entries, "", "  ")
}

// csvSafe neutralises spreadsheet formula injection by prefixing risky cells.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// SortKey returns a time-ordered sort key for the store sk ("<RFC3339Nano>#...").
func SortKey(at time.Time, id string) string {
	return at.UTC().Format(time.RFC3339Nano) + "#" + id
}

// FilterAction returns entries whose action matches (empty = all).
func FilterAction(entries []Entry, action string) []Entry {
	if action == "" {
		return entries
	}
	var out []Entry
	for _, e := range entries {
		if strings.EqualFold(e.Action, action) {
			out = append(out, e)
		}
	}
	return out
}
