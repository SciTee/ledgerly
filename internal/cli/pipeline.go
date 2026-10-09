package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SciTee/ledgerly/internal/categorize"
	"github.com/SciTee/ledgerly/internal/merge"
	"github.com/SciTee/ledgerly/internal/model"
	"github.com/SciTee/ledgerly/internal/parse"
	"github.com/SciTee/ledgerly/internal/profile"
)

// pipelineOptions configures how input files are turned into a timeline.
type pipelineOptions struct {
	profile      *profile.Profile
	account      string // overrides the per-file default when set
	rulesFile    string // user rules file; empty means defaults only
	noCategorize bool
}

// pipelineResult carries the processed timeline plus a few stats worth
// reporting to the user.
type pipelineResult struct {
	Transactions []model.Transaction
	Duplicates   int
}

// run reads every input file with the chosen profile, merges them into one
// timeline, removes rows re-exported across input files and (unless disabled)
// categorizes the result.
func runPipeline(inputs []string, opts pipelineOptions) (*pipelineResult, error) {
	groups := make([][]model.Transaction, 0, len(inputs))
	for _, in := range inputs {
		txs, err := readOne(in, opts)
		if err != nil {
			return nil, err
		}
		groups = append(groups, txs)
	}

	deduped, dups := merge.DedupFiles(groups...)
	timeline := merge.Merge(deduped)

	if !opts.noCategorize {
		rs, err := loadRules(opts.rulesFile)
		if err != nil {
			return nil, err
		}
		rs.Apply(timeline)
	}

	return &pipelineResult{Transactions: timeline, Duplicates: dups}, nil
}

func readOne(path string, opts pipelineOptions) ([]model.Transaction, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tab, err := parse.ReadCSVOpts(f, opts.profile.ReadOptions())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	account := opts.account
	if account == "" {
		account = accountFromFilename(path)
	}
	return opts.profile.Apply(tab, account)
}

func loadRules(path string) (*categorize.RuleSet, error) {
	if path == "" {
		return categorize.Default()
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return categorize.LoadWithDefaults(f)
}

func accountFromFilename(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
