package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI executes the command tree with the given args, capturing stdout.
func runCLI(args ...string) (stdout string, err error) {
	root := NewRootCmd("test")
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), err
}

func sampleFile() string {
	return filepath.Join("testdata", "ing_sample.csv")
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "in.csv")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConvertDedupsAcrossFilesKeepsRepeatsWithinOneFile(t *testing.T) {
	first := writeTemp(t, `Buchung;Valuta;Auftraggeber/Empfänger;Buchungstext;Verwendungszweck;Betrag;Währung
06.02.2026;06.02.2026;SPOTIFY;Lastschrift;Premium Abo;-9,99;EUR
06.02.2026;06.02.2026;Shell Station 4711;Lastschrift;Tanken;-62,30;EUR
`)
	// Re-exports the Spotify row twice (one overlap, one real repeat that only
	// the longer window contains) plus one booking the first file lacks.
	second := writeTemp(t, `Buchung;Valuta;Auftraggeber/Empfänger;Buchungstext;Verwendungszweck;Betrag;Währung
06.02.2026;06.02.2026;SPOTIFY;Lastschrift;Premium Abo;-9,99;EUR
06.02.2026;06.02.2026;SPOTIFY;Lastschrift;Premium Abo;-9,99;EUR
07.02.2026;07.02.2026;ROSSMANN;Lastschrift;Drogerie;-12,45;EUR
`)

	out, err := runCLI("convert", "-p", "ING", first, second)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// header + first file's 2 rows + the later file's surplus (one repeat) and
	// its new row; only the re-exported overlap is dropped.
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5:\n%s", len(lines), out)
	}
	spotify := 0
	for _, l := range lines[1:] {
		if strings.Contains(l, "SPOTIFY") {
			spotify++
		}
	}
	if spotify != 2 {
		t.Errorf("spotify rows = %d, want 2 (both real occurrences kept)", spotify)
	}
}

func TestConvertCSVEndToEnd(t *testing.T) {
	out, err := runCLI("convert", "-p", "ING", sampleFile())
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 { // header + 4 rows
		t.Fatalf("got %d lines, want 5:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "date,value_date,amount") {
		t.Errorf("unexpected header: %q", lines[0])
	}
	// Rows are ordered by date, then by amount; the two 06.02 rows sort by
	// amount (Shell -62.30 before Spotify -9.99).
	wantCategories := []string{"Groceries", "Income", "Transport", "Subscriptions"}
	for i, want := range wantCategories {
		if !strings.Contains(lines[i+1], want) {
			t.Errorf("row %d %q missing category %q", i, lines[i+1], want)
		}
	}
}

func TestConvertJSONAutoDetect(t *testing.T) {
	out, err := runCLI("convert", "--auto-detect", "-f", "json", sampleFile())
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if !strings.Contains(out, `"category": "Transport"`) {
		t.Errorf("expected Transport category in JSON output:\n%s", out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Errorf("output is not a JSON array:\n%s", out)
	}
}

func TestSummaryEndToEnd(t *testing.T) {
	out, err := runCLI("summary", "-p", "ING", sampleFile())
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	for _, want := range []string{"By category:", "Totals:", "Net"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestProfilesList(t *testing.T) {
	out, err := runCLI("profiles", "list")
	if err != nil {
		t.Fatalf("profiles list: %v", err)
	}
	if !strings.Contains(out, "ING") {
		t.Errorf("profiles list missing ING:\n%s", out)
	}
}

func TestConvertUnknownProfileErrors(t *testing.T) {
	if _, err := runCLI("convert", "-p", "DoesNotExist", sampleFile()); err == nil {
		t.Error("expected error for unknown profile")
	}
}

func TestConvertWithoutProfileErrors(t *testing.T) {
	if _, err := runCLI("convert", sampleFile()); err == nil {
		t.Error("expected error when neither --profile nor --auto-detect is given")
	}
}
