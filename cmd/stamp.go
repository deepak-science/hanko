package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmallubhotla/hanko/internal/config"
	"github.com/dmallubhotla/hanko/internal/stamper"
	"github.com/dmallubhotla/hanko/internal/version"
	"github.com/spf13/cobra"
)

var stampDryRun bool

var stampCmd = &cobra.Command{
	Use:     "stamp [format]",
	Short:   "Apply declared stamp-targets, optionally filtered by format",
	GroupID: "stamp",
	Long: `Reads ` + "`stamp-targets:`" + ` from ` + "`.hanko.yaml`" + ` and updates each declared file to the current computed version.

With a positional ` + "`[format]`" + ` argument, only targets whose ` + "`format:`" + ` matches are stamped — useful as a build-step that touches only one kind of file (` + "`hanko stamp helm`" + `, ` + "`hanko stamp nix`" + `, etc.). If no targets match the filter, hanko errors rather than silently no-op.

For build-time emitters that print to stdout (Go ldflags, container tags, OCI labels), see ` + "`hanko version go-ldflags`" + ` and ` + "`hanko version docker tags|labels`" + `.

` + "`--dry-run`" + ` prints per-target before/after summaries without writing.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := ""
		if len(args) == 1 {
			filter = args[0]
		}
		return runStampTargets(filter)
	},
}

func runStampTargets(filter string) error {
	cfg, err := config.Load(repoPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if len(cfg.StampTargets) == 0 {
		return fmt.Errorf("no `stamp-targets:` declared in %s — add targets to .hanko.yaml", configSourceOrDefault(cfg))
	}
	targets := cfg.StampTargets
	if filter != "" {
		targets = filterTargetsByFormat(cfg.StampTargets, filter)
		if len(targets) == 0 {
			return fmt.Errorf("no `stamp-targets:` with format %q in %s", filter, configSourceOrDefault(cfg))
		}
	}
	v, err := resolveVersion("")
	if err != nil {
		return err
	}
	return applyStampTargets(targets, v, stampDryRun)
}

// filterTargetsByFormat returns the subset of targets whose Format matches
// the given filter exactly. No alias collapsing (e.g. yaml/yml are distinct
// here, matching whatever the user wrote in `.hanko.yaml`).
func filterTargetsByFormat(targets []config.StampTarget, format string) []config.StampTarget {
	var out []config.StampTarget
	for _, t := range targets {
		if t.Format == format {
			out = append(out, t)
		}
	}
	return out
}

func applyStampTargets(targets []config.StampTarget, v version.Version, dryRun bool) error {
	for _, t := range targets {
		keys := t.EffectiveKeys()
		// Some engines (plain, helm) determine their own keys.
		if !stamper.FormatHasFixedKeys(t.Format) && len(keys) == 0 {
			return fmt.Errorf("stamp-target %q (%s): `key` or `keys` required", t.Path, t.Format)
		}
		abs := t.Path
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(repoPath, t.Path)
		}
		orig, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("read %s: %w", t.Path, err)
		}
		updated, desc, err := stamper.Stamp(t.Format, orig, keys, v.SemVer)
		if err != nil {
			return fmt.Errorf("%s: %w", t.Path, err)
		}
		if dryRun {
			fmt.Printf("%s (%s): %s\n", t.Path, t.Format, desc)
			continue
		}
		if !bytes.Equal(orig, updated) {
			if err := os.WriteFile(abs, updated, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", t.Path, err)
			}
		}
		fmt.Printf("%s: %s\n", t.Path, desc)
	}
	return nil
}

func configSourceOrDefault(cfg *config.Config) string {
	if cfg.Source != "" {
		return cfg.Source
	}
	return "(no .hanko.yaml; using defaults)"
}

func init() {
	stampCmd.Flags().BoolVar(&stampDryRun, "dry-run", false, "print per-target changes without writing")
	rootCmd.AddCommand(stampCmd)
}
