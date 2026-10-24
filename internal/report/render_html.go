package report

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

var htmlTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"upper": strings.ToUpper,
	"pct":   func(f float64) string { return fmt.Sprintf("%.0f", f) },
	"add1":  func(i int) int { return i + 1 },
}).Parse(htmlSource))

// RenderHTML renders a self-contained, print-ready HTML report. All text is
// escaped by html/template; no external requests.
func RenderHTML(d Data) []byte {
	var buf bytes.Buffer
	if err := htmlTmpl.Execute(&buf, d); err != nil {
		return []byte("<html><body>report render error: " + template.HTMLEscapeString(err.Error()) + "</body></html>")
	}
	return buf.Bytes()
}

const htmlSource = `<!doctype html>
<html lang="en-GB"><head><meta charset="utf-8">
<title>Security Posture Report</title>
<style>
 body{font:14px/1.5 -apple-system,Segoe UI,Roboto,sans-serif;color:#1a2332;max-width:900px;margin:0 auto;padding:32px}
 h1{font-size:26px} h2{font-size:19px;border-bottom:1px solid #ddd;padding-bottom:4px;margin-top:28px} h3{font-size:15px}
 table{border-collapse:collapse;width:100%;margin:10px 0} th,td{border:1px solid #ddd;padding:6px 8px;text-align:left;font-size:13px}
 th{background:#f4f6fa} .score{font-size:40px;font-weight:800} .wm{background:#fde;color:#900;padding:8px;border-radius:6px;font-weight:700}
 .sev-CRITICAL{color:#b00020;font-weight:700}.sev-HIGH{color:#c25700;font-weight:700}.sev-MEDIUM{color:#8a6d00}.sev-LOW{color:#2d6}
 .logo{max-height:60px}
 @media print{h2{page-break-before:always}h2:first-of-type{page-break-before:avoid}body{max-width:none}}
</style></head><body>
{{if .Watermark}}<div class="wm">{{.Watermark}}</div>{{end}}
{{if .LogoDataURI}}<img class="logo" src="{{.LogoDataURI}}" alt="logo">{{end}}
<h1>Security Posture Report — {{if .ClientName}}{{.ClientName}}{{else}}{{.Account}}{{end}}</h1>
<p>Prepared by <b>{{.MSPName}}</b> · {{.GeneratedAt.Format "2 Jan 2006"}} · scope: {{.Scope}} · status: {{.Status}} (v{{.Version}})</p>

<h2>Executive summary</h2><p>{{.Summary}}</p>

<h2>Posture score</h2><div class="score">{{.Score}} / 100</div><p>{{.Rating}}</p>

<h2>What changed since the last scan</h2>
<ul><li>New: {{.Changes.NewCount}}</li><li>Fixed: {{.Changes.FixedCount}}</li><li>Regressed: {{.Changes.RegressedCount}}</li></ul>

<h2>Fix these first</h2>
{{if .TopFixes}}<table><tr><th>#</th><th>Severity</th><th>Risk</th><th>Finding</th><th>Region</th></tr>
{{range $i,$t := .TopFixes}}<tr><td>{{add1 $i}}</td><td class="sev-{{$t.Finding.Severity}}">{{$t.Finding.Severity}}</td><td>{{pct $t.Finding.RiskScore}}</td><td>{{$t.Finding.Title}}</td><td>{{$t.Finding.Region}}</td></tr>{{end}}
</table>{{else}}<p>No open findings.</p>{{end}}

{{if .Paths}}<h2>Attack paths</h2>
{{range .Paths}}<h3>{{.Kind}} — {{.Title}} ({{.Severity}})</h3><ul>{{range .Evidence}}<li>{{.}}</li>{{end}}</ul><p><b>Break the chain:</b> {{.BreakHint}}</p>{{end}}
{{end}}

<h2>Findings by service</h2>
{{range .ByService}}<h3>{{upper .Service}} ({{len .Findings}})</h3>
<table><tr><th>Severity</th><th>Risk</th><th>Status</th><th>Finding</th></tr>
{{range .Findings}}<tr><td class="sev-{{.Severity}}">{{.Severity}}</td><td>{{pct .RiskScore}}</td><td>{{.Status}}</td><td>{{.Title}}</td></tr>{{end}}
</table>{{end}}

{{if .AcceptedRisks}}<h2>Accepted risks</h2><table><tr><th>Reason</th><th>Owner</th><th>Expires</th></tr>
{{range .AcceptedRisks}}<tr><td>{{.Reason}}</td><td>{{.Owner}}</td><td>{{.ExpiresAt.Format "2 Jan 2006"}}</td></tr>{{end}}</table>{{end}}

{{if .CIS}}<h2>CIS-mapped controls</h2><table><tr><th>Control</th><th>Pass</th><th>Fail</th><th>Accepted</th></tr>
{{range .CIS}}<tr><td>{{.Control}}</td><td>{{.Pass}}</td><td>{{.Fail}}</td><td>{{.Accepted}}</td></tr>{{end}}</table>{{end}}

<h2>Scope and method</h2><ul><li>Account: {{.Account}}</li><li>Regions: {{range .Regions}}{{.}} {{end}}</li>
{{range .SourcesRun}}<li>Source {{.Source}}: ran={{.Ran}} count={{.Count}}</li>{{end}}
<li>Read-only, point-in-time.</li></ul>

<h2>Review record</h2><ul><li>Reviewer: {{.Reviewer}}</li><li>Status: {{.Status}}</li>{{if .ReviewNote}}<li>Note: {{.ReviewNote}}</li>{{end}}</ul>
</body></html>`
