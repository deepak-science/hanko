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
	Use:     "stamp",
	Short:   "Stamp the computed version onto declared targets (or use a subcommand for one-offs)",
	GroupID: "stamp",
	Long: `With no subcommand, reads ` + "`stamp-targets:`" + ` from ` + "`.hanko.yaml`" + ` and updates each declared file to the current computed version.

Subcommands (` + "`go-ldflags`" + `, ` + "`docker tags`" + `, ` + "`docker labels`" + `, ` + "`helm`" + `, ` + "`nix`" + `) remain for one-off use without config.

` + "`--dry-run`" + ` prints per-target before/after summaries without writing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Only the no-args call falls into this RunE — subcommands have their own.
		return runStampTargets()
	},
}

func runStampTargets() error {
	cfg, err := config.Load(repoPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if len(cfg.StampTargets) == 0 {
		return fmt.Errorf("no `stamp-targets:` declared in %s — use a `hanko stamp <format>` subcommand for one-off stamping, or add targets to .hanko.yaml", configSourceOrDefault(cfg))
	}
	v, err := resolveVersion("")
	if err != nil {
		return err
	}
	return applyStampTargets(cfg.StampTargets, v, stampDryRun)
}

func applyStampTargets(targets []config.StampTarget, v version.Version, dryRun bool) error {
	for _, t := range targets {
		keys := t.EffectiveKeys()
		// "plain" format ignores keys; everything else requires at least one.
		if t.Format != "plain" && t.Format != "text" && len(keys) == 0 {
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
