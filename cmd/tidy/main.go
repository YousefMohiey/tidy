package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/verhafter/tidy/internal/config"
	"github.com/verhafter/tidy/internal/dedup"
	"github.com/verhafter/tidy/internal/organizer"
	"github.com/verhafter/tidy/internal/paths"
	"github.com/verhafter/tidy/internal/tui"
	"github.com/verhafter/tidy/internal/watcher"
)

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
}

// rootCmd builds and returns the root cobra command with all subcommands attached.
// When invoked without a subcommand, launches the interactive dashboard directly.
func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tidy",
		Short: "Smart file organizer",
		Long:  "tidy automatically organizes files into categorized directories\nbased on extension, MIME type, and filename patterns.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchDashboard("", "")
		},
	}

	root.AddCommand(
		newOrganizeCmd(),
		newWatchCmd(),
		newUndoCmd(),
		newStatusCmd(),
		newDedupCmd(),
		newDashboardCmd(),
	)

	return root
}

func launchDashboard(journalPath, scanDir string) error {
	cfg, err := loadConfig("")
	if err != nil {
		cfg = config.Default()
	}

	data := tui.DashboardData{
		Config: cfg,
	}

	jPath, err := resolveJournalPath(journalPath)
	if err == nil {
		journal, err := organizer.LoadJournal(jPath)
		if err == nil {
			data.Journal = journal
			data.SourceDir = journal.SourceDir
		}
	}

	if scanDir != "" {
		scanner := dedup.NewScanner()
		result, err := scanner.Scan(scanDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%swarning: dedup scan failed: %s%s\n", colorYellow, err, colorReset)
		} else {
			data.DedupScan = result
			if data.SourceDir == "" {
				data.SourceDir = scanDir
			}
		}
	}

	return tui.Run(data)
}

// --- organize command --------------------------------------------------------

func newOrganizeCmd() *cobra.Command {
	var (
		dryRun     bool
		configPath string
	)

	cmd := &cobra.Command{
		Use:   "organize <directory>",
		Short: "Organize files in a directory",
		Long:  "Scan the given directory and move files into categorized subdirectories\nbased on the configured rules.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]

			cfg, err := loadConfig(configPath)
			if err != nil {
				return fmt.Errorf("%sconfiguration error: %s%s", colorRed, err, colorReset)
			}

			opts := organizer.Options{DryRun: dryRun}
			org := organizer.New(cfg, opts)

			result, err := org.Organize(dir)
			if err != nil {
				return fmt.Errorf("%sorganize failed: %s%s", colorRed, err, colorReset)
			}

			printOrganizeResult(result)

			if len(result.Errors) > 0 {
				return fmt.Errorf("%s%d file(s) had errors%s", colorRed, len(result.Errors), colorReset)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without moving files")
	cmd.Flags().StringVar(&configPath, "config", "", "path to YAML config file")

	return cmd
}

// printOrganizeResult formats and prints the organize result to stdout.
func printOrganizeResult(result *organizer.Result) {
	if result.DryRun {
		fmt.Fprintf(os.Stdout, "%s[dry-run] Preview of planned moves:%s\n", colorYellow, colorReset)
		for _, m := range result.Moves {
			fmt.Fprintf(os.Stdout, "  %s%s%s → %s%s%s\n",
				colorYellow, m.Source, colorReset,
				colorYellow, m.Destination, colorReset,
			)
		}
		fmt.Fprintf(os.Stdout, "\n%sWould organize %d files (%d moves, %d skipped)%s\n",
			colorYellow, result.FilesProcessed, result.FilesMoved, result.FilesSkipped, colorReset,
		)
		return
	}

	fmt.Fprintf(os.Stdout, "%sOrganized %d files (%d moved, %d skipped)%s\n",
		colorGreen, result.FilesProcessed, result.FilesMoved, result.FilesSkipped, colorReset,
	)

	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stdout, "\n%sErrors:%s\n", colorRed, colorReset)
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stdout, "  %s✗ %s%s\n", colorRed, e, colorReset)
		}
	}
}

// --- watch command -----------------------------------------------------------

func newWatchCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "watch <directory>",
		Short: "Watch a directory and organize new files automatically",
		Long:  "Continuously monitor the given directory for new or modified files\nand organize them as they appear. Press Ctrl+C to stop.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]

			cfg, err := loadConfig(configPath)
			if err != nil {
				return fmt.Errorf("%sconfiguration error: %s%s", colorRed, err, colorReset)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(sigCh)
			go func() {
				select {
				case <-sigCh:
					cancel()
				case <-ctx.Done():
				}
			}()

			w := watcher.New(dir, cfg, organizer.Options{})

			fmt.Fprintf(os.Stdout, "%sWatching %s... (Ctrl+C to stop)%s\n", colorGreen, dir, colorReset)

			if err := w.Watch(ctx); err != nil && ctx.Err() == nil {
				return fmt.Errorf("%swatch failed: %s%s", colorRed, err, colorReset)
			}

			fmt.Fprintf(os.Stdout, "\n%sStopped watching.%s\n", colorGreen, colorReset)
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "path to YAML config file")

	return cmd
}

// --- undo command ------------------------------------------------------------

func newUndoCmd() *cobra.Command {
	var journalPath string

	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Undo the last organize operation",
		Long:  "Restore files to their original locations by reversing the operations\nrecorded in the journal file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jPath, err := resolveJournalPath(journalPath)
			if err != nil {
				return fmt.Errorf("%s%s%s", colorRed, err, colorReset)
			}

			journal, err := organizer.LoadJournal(jPath)
			if err != nil {
				return fmt.Errorf("%sfailed to load journal: %s%s", colorRed, err, colorReset)
			}

			fmt.Fprintf(os.Stdout, "Undoing %d operations from journal...\n", len(journal.Operations))

			restored, err := journal.Undo()
			if err != nil {
				return fmt.Errorf("%sundo failed: %s%s", colorRed, err, colorReset)
			}

			fmt.Fprintf(os.Stdout, "%sRestored %d files.%s\n", colorGreen, restored, colorReset)

			// Remove the journal file after successful undo.
			if err := os.Remove(jPath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stdout, "%swarning: could not remove journal file: %s%s\n", colorYellow, err, colorReset)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&journalPath, "journal", "", "path to journal file (default: platform-specific data dir)")

	return cmd
}

// --- status command ----------------------------------------------------------

func newStatusCmd() *cobra.Command {
	var journalPath string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the last organize operation status",
		Long:  "Display a summary of the most recent organize operation,\nincluding timestamp, file count, and individual moves.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jPath, err := resolveJournalPath(journalPath)
			if err != nil {
				return fmt.Errorf("%s%s%s", colorRed, err, colorReset)
			}

			journal, err := organizer.LoadJournal(jPath)
			if err != nil {
				return fmt.Errorf("%sfailed to load journal: %s%s", colorRed, err, colorReset)
			}

			fmt.Fprintf(os.Stdout, "%sLast organized:%s %s\n", colorGreen, colorReset, journal.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Fprintf(os.Stdout, "%sOperations:%s    %d\n", colorGreen, colorReset, len(journal.Operations))
			fmt.Fprintf(os.Stdout, "%sSource:%s        %s\n", colorGreen, colorReset, journal.SourceDir)

			if len(journal.Operations) > 0 {
				fmt.Fprintln(os.Stdout)
				for _, op := range journal.Operations {
					fmt.Fprintf(os.Stdout, "  %s → %s\n", op.Source, op.Destination)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&journalPath, "journal", "", "path to journal file (default: platform-specific data dir)")

	return cmd
}

// --- dedup command -----------------------------------------------------------

func newDedupCmd() *cobra.Command {
	var (
		launchDashboard bool
		outputJSON      bool
		configPath      string
	)

	cmd := &cobra.Command{
		Use:   "dedup <directory> [directory...]",
		Short: "Find duplicate files by content hash",
		Long:  "Scan one or more directories recursively and group files with identical content.\nUses size-first optimization and streaming SHA256 for efficiency.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scanner := dedup.NewScanner()

			result, err := scanner.Scan(args...)
			if err != nil {
				return fmt.Errorf("%sdedup failed: %s%s", colorRed, err, colorReset)
			}

			if outputJSON {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("%sfailed to marshal JSON: %s%s", colorRed, err, colorReset)
				}
				fmt.Fprintln(os.Stdout, string(data))
				return nil
			}

			fmt.Fprintf(os.Stdout, "%sScanned:%s       %d files across %d directories\n",
				colorGreen, colorReset, result.TotalFiles, len(result.ScannedDirs))
			fmt.Fprintf(os.Stdout, "%sUnique:%s        %d files\n",
				colorGreen, colorReset, result.UniqueFiles)
			fmt.Fprintf(os.Stdout, "%sDuplicates:%s    %d groups\n",
				colorGreen, colorReset, len(result.DuplicateGroups))
			fmt.Fprintf(os.Stdout, "%sWasted space:%s  %s\n",
				colorGreen, colorReset, dedup.FormatSize(result.WastedBytes))

			if len(result.DuplicateGroups) > 0 {
				fmt.Fprintln(os.Stdout)
				for i, g := range result.DuplicateGroups {
					copies := len(g.Files)
					wasted := int64(copies-1) * g.Size
					fmt.Fprintf(os.Stdout, "%sGroup %d%s (%s, %d copies, %s wasted):\n",
						colorYellow, i+1, colorReset, dedup.FormatSize(g.Size), copies, dedup.FormatSize(wasted))
					for _, f := range g.Files {
						fmt.Fprintf(os.Stdout, "  %s\n", f)
					}
					if i < len(result.DuplicateGroups)-1 {
						fmt.Fprintln(os.Stdout)
					}
				}
			}

			if launchDashboard {
				fmt.Fprintln(os.Stdout)
				fmt.Fprintf(os.Stdout, "%sLaunching dashboard...%s\n", colorGreen, colorReset)
				cfg, cfgErr := loadConfig(configPath)
				if cfgErr != nil {
					cfg = config.Default()
				}
				jPath, _ := resolveJournalPath("")
				journal, _ := organizer.LoadJournal(jPath)
				data := tui.DashboardData{
					Journal:   journal,
					DedupScan: result,
					SourceDir: args[0],
					Config:    cfg,
				}
				return tui.Run(data)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&launchDashboard, "dashboard", false, "launch TUI dashboard after scan")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "output results as JSON")
	cmd.Flags().StringVar(&configPath, "config", "", "path to YAML config file (for dashboard)")

	return cmd
}

// --- dashboard command ------------------------------------------------------

func newDashboardCmd() *cobra.Command {
	var (
		journalPath string
		scanDir     string
	)

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch interactive TUI dashboard",
		Long:  "Display an interactive terminal dashboard showing recent operations\nand duplicate file analysis.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchDashboard(journalPath, scanDir)
		},
	}

	cmd.Flags().StringVar(&journalPath, "journal", "", "path to journal file (default: ~/.local/share/tidy/journal.json)")
	cmd.Flags().StringVar(&scanDir, "dir", "", "directory to scan for duplicates")

	return cmd
}

// --- helpers -----------------------------------------------------------------

// loadConfig resolves the configuration using the following precedence:
//  1. Explicit --config flag
//  2. config.yaml in the current working directory
//  3. Built-in defaults
func loadConfig(path string) (*config.Config, error) {
	if path != "" {
		return config.Load(path)
	}

	// Try config.yaml in current directory.
	if _, err := os.Stat("config.yaml"); err == nil {
		return config.Load("config.yaml")
	}

	// Fall back to defaults.
	return config.Default(), nil
}

// resolveJournalPath returns the journal file path, using the platform-specific
// default location when none is specified.
func resolveJournalPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	return paths.JournalPath(), nil
}
