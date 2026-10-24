package audit

import (
	"strings"
	"testing"
	"time"
)

func TestCSVEscapesFormulaInjection(t *testing.T) {
	entries := []Entry{{At: time.Now(), Actor: "=cmd()", Action: ScanFull, Detail: "+danger"}}
	b, err := ExportCSV(entries)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "'=cmd()") || !strings.Contains(s, "'+danger") {
		t.Fatalf("formula injection not escaped: %s", s)
	}
}

func TestExportJSON(t *testing.T) {
	b, err := ExportJSON([]Entry{{Action: LoginSuccess}})
	if err != nil || !strings.Contains(string(b), "login.success") {
		t.Fatalf("json export: %v %s", err, b)
	}
}

func TestFilterAction(t *testing.T) {
	es := []Entry{{Action: ScanFull}, {Action: LoginSuccess}}
	if len(FilterAction(es, ScanFull)) != 1 {
		t.Fatal("filter should keep one")
	}
	if len(FilterAction(es, "")) != 2 {
		t.Fatal("empty filter keeps all")
	}
}
