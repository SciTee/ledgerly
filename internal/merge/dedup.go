package merge

import (
	"strconv"
	"strings"

	"github.com/SciTee/ledgerly/internal/model"
)

// DedupFiles removes transactions that a later input file re-exports from an
// earlier one, keeping every row within a single file. Two rows are considered
// the same when their booking date, amount, payee and purpose match — the
// signature of a row that shows up twice because two exports covered an
// overlapping period. Files are processed in order and compared as multisets:
// for each identity a later file keeps its surplus beyond what earlier files
// already contributed, so a longer export whose window genuinely contains the
// same real booking twice still lands once per real occurrence. Identical rows
// inside one file are legitimate bookings (two same-day purchases, split
// payments) and always survive. It returns the kept transactions in input
// order and the number of rows dropped.
func DedupFiles(groups ...[]model.Transaction) (kept []model.Transaction, removed int) {
	prior := make(map[string]int)
	kept = make([]model.Transaction, 0)
	for _, g := range groups {
		current := make(map[string]int, len(g))
		for _, tx := range g {
			current[dedupKey(tx)]++
		}
		budget := make(map[string]int, len(current))
		for k, n := range current {
			if surplus := n - prior[k]; surplus > 0 {
				budget[k] = surplus
			}
		}
		for _, tx := range g {
			k := dedupKey(tx)
			if budget[k] > 0 {
				budget[k]--
				kept = append(kept, tx)
				continue
			}
			removed++
		}
		for k, n := range current {
			prior[k] += n
		}
	}
	return kept, removed
}

// dedupKey builds the identity string used to spot duplicates. The account is
// deliberately excluded so the same transaction exported from two overlapping
// statements of one account collapses, while normalisation (lower-case,
// collapsed whitespace) absorbs cosmetic differences between exports.
func dedupKey(tx model.Transaction) string {
	var b strings.Builder
	b.WriteString(tx.Date.String())
	b.WriteByte('|')
	b.WriteString(strconv.FormatInt(int64(tx.Amount), 10))
	b.WriteByte('|')
	b.WriteString(normalize(tx.Payee))
	b.WriteByte('|')
	b.WriteString(normalize(tx.Purpose))
	return b.String()
}

func normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
