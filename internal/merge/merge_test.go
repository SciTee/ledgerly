package merge

import (
	"testing"
	"time"

	"github.com/SciTee/ledgerly/internal/model"
)

func mk(day int, amount model.Money, payee, purpose, account string) model.Transaction {
	return model.Transaction{
		Date:    model.NewDate(2026, time.January, day),
		Amount:  amount,
		Payee:   payee,
		Purpose: purpose,
		Account: account,
	}
}

func TestMergeChronological(t *testing.T) {
	giro := []model.Transaction{
		mk(10, -500, "REWE", "Einkauf", "giro"),
		mk(3, 245000, "Arbeitgeber", "Gehalt", "giro"),
	}
	credit := []model.Transaction{
		mk(7, -1299, "Netflix", "Abo", "credit"),
	}

	got := Merge(giro, credit)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	wantDays := []int{3, 7, 10}
	for i, d := range wantDays {
		if got[i].Date.Day() != d {
			t.Errorf("position %d: day = %d, want %d", i, got[i].Date.Day(), d)
		}
	}
}

func TestMergeDoesNotMutateInputs(t *testing.T) {
	a := []model.Transaction{mk(5, 100, "B", "", "x"), mk(1, 100, "A", "", "x")}
	_ = Merge(a)
	if a[0].Date.Day() != 5 {
		t.Errorf("input slice was reordered: first day = %d, want 5", a[0].Date.Day())
	}
}

func TestDedupFilesRemovesOverlap(t *testing.T) {
	// Two exports of the same account that overlap on the 7th and 8th.
	exportA := []model.Transaction{
		mk(6, -500, "REWE", "Einkauf", "giro"),
		mk(7, -1299, "Netflix", "Abo Januar", "giro"),
		mk(8, 245000, "Arbeitgeber", "Gehalt", "giro"),
	}
	exportB := []model.Transaction{
		mk(7, -1299, "Netflix", "Abo Januar", "giro"),  // re-exported from A
		mk(8, 245000, "Arbeitgeber", "Gehalt", "giro"), // re-exported from A
		mk(9, -4200, "Shell", "Tanken", "giro"),
	}

	kept, removed := DedupFiles(exportA, exportB)
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
	if len(kept) != 4 {
		t.Errorf("kept = %d, want 4", len(kept))
	}
}

func TestDedupFilesKeepsRepeatedRowsWithinOneFile(t *testing.T) {
	// A single bank export can legitimately contain identical bookings —
	// two same-day purchases of the same amount, split payments, batch-processed
	// repeats. They are real money and must all survive.
	txs := []model.Transaction{
		mk(4, -350, "Bakery A", "Coffee", "giro"),
		mk(4, -350, "Bakery A", "Coffee", "giro"),
		mk(4, -350, "Bakery A", "Coffee", "giro"),
	}
	kept, removed := DedupFiles(txs)
	if removed != 0 || len(kept) != 3 {
		t.Errorf("repeated rows within one file collapsed: kept=%d removed=%d", len(kept), removed)
	}
}

func TestDedupFilesKeepsSurplusRepeatsInLaterFile(t *testing.T) {
	// The later export covers a longer window that genuinely contains the
	// same real booking twice; only the one occurrence already covered by the
	// earlier file may drop.
	exportA := []model.Transaction{
		mk(4, -350, "Bakery A", "Coffee", "giro"),
	}
	exportB := []model.Transaction{
		mk(4, -350, "Bakery A", "Coffee", "giro"),
		mk(4, -350, "Bakery A", "Coffee", "giro"),
		mk(5, -1200, "Apotheke", "Hustensaft", "giro"),
	}

	kept, removed := DedupFiles(exportA, exportB)
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if len(kept) != 3 {
		t.Errorf("kept = %d, want 3", len(kept))
	}
}

func TestDedupFilesDropsEverythingWhenSameFileIsGivenTwice(t *testing.T) {
	export := []model.Transaction{
		mk(4, -350, "Bakery A", "Coffee", "giro"),
		mk(5, -1200, "Apotheke", "Hustensaft", "giro"),
	}

	kept, removed := DedupFiles(export, export)
	if removed != 2 || len(kept) != 2 {
		t.Errorf("same file twice: kept=%d removed=%d, want 2/2", len(kept), removed)
	}
}

func TestDedupKeepsDistinctSameDay(t *testing.T) {
	txs := []model.Transaction{
		mk(4, -350, "Bakery A", "Coffee", "giro"),
		mk(4, -350, "Bakery B", "Coffee", "giro"),
	}
	kept, removed := DedupFiles(txs)
	if removed != 0 || len(kept) != 2 {
		t.Errorf("distinct rows collapsed: kept=%d removed=%d", len(kept), removed)
	}
}
