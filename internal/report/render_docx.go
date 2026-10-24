package report

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

// RenderDOCX builds a minimal, valid WordprocessingML (.docx) package with
// archive/zip. Opens in Word and LibreOffice without a repair prompt.
func RenderDOCX(d Data) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	add := func(name, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(content))
		return err
	}

	if err := add("[Content_Types].xml", contentTypesXML); err != nil {
		return nil, err
	}
	if err := add("_rels/.rels", relsXML); err != nil {
		return nil, err
	}
	if err := add("docProps/core.xml", coreXML(d)); err != nil {
		return nil, err
	}
	if err := add("word/_rels/document.xml.rels", docRelsXML); err != nil {
		return nil, err
	}
	if err := add("word/styles.xml", stylesXML); err != nil {
		return nil, err
	}
	if err := add("word/document.xml", documentXML(d)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
</Types>`

const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
</Relationships>`

const docRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:sz w:val="22"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:pPr><w:spacing w:after="120"/></w:pPr><w:rPr><w:b/><w:sz w:val="52"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:pPr><w:spacing w:before="240" w:after="80"/></w:pPr><w:rPr><w:b/><w:sz w:val="32"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:pPr><w:spacing w:before="160" w:after="60"/></w:pPr><w:rPr><w:b/><w:sz w:val="26"/></w:rPr></w:style>
</w:styles>`

func coreXML(d Data) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:title>` + xmlEsc("Security Posture Report — "+d.ClientName) + `</dc:title>
<dc:creator>` + xmlEsc(d.MSPName) + `</dc:creator>
</cp:coreProperties>`
}

func documentXML(d Data) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)

	if d.Watermark != "" {
		b.WriteString(para("Heading2", d.Watermark))
	}
	title := d.ClientName
	if title == "" {
		title = d.Account
	}
	b.WriteString(paraStyle("Title", "Security Posture Report — "+title))
	b.WriteString(para("Normal", fmt.Sprintf("Prepared by %s · %s · scope: %s · status: %s (v%d)",
		d.MSPName, d.GeneratedAt.Format("2 Jan 2006"), d.Scope, d.Status, d.Version)))

	b.WriteString(heading("Executive summary"))
	b.WriteString(para("Normal", d.Summary))

	b.WriteString(heading("Posture score"))
	b.WriteString(para("Normal", fmt.Sprintf("%d / 100 — %s", d.Score, d.Rating)))

	b.WriteString(heading("What changed since the last scan"))
	b.WriteString(bullet(fmt.Sprintf("New: %d", d.Changes.NewCount)))
	b.WriteString(bullet(fmt.Sprintf("Fixed: %d", d.Changes.FixedCount)))
	b.WriteString(bullet(fmt.Sprintf("Regressed: %d", d.Changes.RegressedCount)))

	b.WriteString(heading("Fix these first"))
	if len(d.TopFixes) == 0 {
		b.WriteString(para("Normal", "No open findings."))
	} else {
		rows := [][]string{{"#", "Severity", "Risk", "Finding", "Region"}}
		for i, tf := range d.TopFixes {
			rows = append(rows, []string{
				fmt.Sprintf("%d", i+1), string(tf.Finding.Severity),
				fmt.Sprintf("%.0f", tf.Finding.RiskScore), tf.Finding.Title, tf.Finding.Region,
			})
		}
		b.WriteString(table(rows))
	}

	if len(d.Paths) > 0 {
		b.WriteString(heading("Attack paths"))
		for _, p := range d.Paths {
			b.WriteString(para("Heading2", fmt.Sprintf("%s — %s (%s)", p.Kind, p.Title, p.Severity)))
			for _, e := range p.Evidence {
				b.WriteString(bullet(e))
			}
			b.WriteString(para("Normal", "Break the chain: "+p.BreakHint))
		}
	}

	b.WriteString(heading("Findings by service"))
	for _, g := range d.ByService {
		b.WriteString(para("Heading2", fmt.Sprintf("%s (%d)", strings.ToUpper(g.Service), len(g.Findings))))
		rows := [][]string{{"Severity", "Risk", "Status", "Finding"}}
		for _, f := range g.Findings {
			rows = append(rows, []string{string(f.Severity), fmt.Sprintf("%.0f", f.RiskScore), string(f.Status), f.Title})
		}
		b.WriteString(table(rows))
	}

	if len(d.AcceptedRisks) > 0 {
		b.WriteString(heading("Accepted risks"))
		rows := [][]string{{"Reason", "Owner", "Expires"}}
		for _, r := range d.AcceptedRisks {
			rows = append(rows, []string{r.Reason, r.Owner, r.ExpiresAt.Format("2 Jan 2006")})
		}
		b.WriteString(table(rows))
	}

	if len(d.CIS) > 0 {
		b.WriteString(heading("CIS-mapped controls"))
		rows := [][]string{{"Control", "Pass", "Fail", "Accepted"}}
		for _, c := range d.CIS {
			rows = append(rows, []string{c.Control, fmt.Sprintf("%d", c.Pass), fmt.Sprintf("%d", c.Fail), fmt.Sprintf("%d", c.Accepted)})
		}
		b.WriteString(table(rows))
	}

	b.WriteString(heading("Scope and method"))
	b.WriteString(bullet("Account: " + d.Account))
	b.WriteString(bullet("Regions: " + strings.Join(d.Regions, ", ")))
	for _, s := range d.SourcesRun {
		b.WriteString(bullet(fmt.Sprintf("Source %s: ran=%v count=%d", s.Source, s.Ran, s.Count)))
	}
	b.WriteString(bullet("Read-only, point-in-time."))

	b.WriteString(heading("Review record"))
	b.WriteString(bullet("Reviewer: " + d.Reviewer))
	b.WriteString(bullet("Status: " + d.Status))
	if d.ReviewNote != "" {
		b.WriteString(bullet("Note: " + d.ReviewNote))
	}

	// section properties with language
	b.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/></w:sectPr>`)
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func heading(text string) string { return paraStyle("Heading1", text) }

func para(style, text string) string { return paraStyle(style, text) }

func paraStyle(style, text string) string {
	return `<w:p><w:pPr><w:pStyle w:val="` + style + `"/></w:pPr><w:r><w:t xml:space="preserve">` + xmlEsc(text) + `</w:t></w:r></w:p>`
}

func bullet(text string) string {
	return `<w:p><w:pPr><w:ind w:left="360"/></w:pPr><w:r><w:t xml:space="preserve">` + xmlEsc("• "+text) + `</w:t></w:r></w:p>`
}

func table(rows [][]string) string {
	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblBorders>`)
	for _, edge := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		b.WriteString(`<w:` + edge + ` w:val="single" w:sz="4" w:color="999999"/>`)
	}
	b.WriteString(`</w:tblBorders></w:tblPr>`)
	for ri, row := range rows {
		b.WriteString(`<w:tr>`)
		for _, cell := range row {
			shade := ""
			if ri == 0 {
				shade = `<w:shd w:val="clear" w:fill="F0F2F5"/>`
			}
			b.WriteString(`<w:tc><w:tcPr>` + shade + `</w:tcPr><w:p><w:r><w:t xml:space="preserve">` + xmlEsc(cell) + `</w:t></w:r></w:p></w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
	// a trailing empty paragraph is required after a table
	b.WriteString(`<w:p/>`)
	return b.String()
}

// xmlEsc escapes XML special chars and strips characters invalid in XML 1.0.
func xmlEsc(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r != 0x9 && r != 0xA && r != 0xD && (r < 0x20 || r == 0xFFFE || r == 0xFFFF) {
			continue
		}
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
