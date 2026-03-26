package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/txperl/snipfly/internal/platform"
	"github.com/txperl/snipfly/internal/runner"
	"github.com/txperl/snipfly/internal/snippet"
	"github.com/txperl/snipfly/internal/tui"

	tea "charm.land/bubbletea/v2"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"

	autoDetectSubDir = ".snipfly"
)

// Run executes the CLI application.
func Run() error {
	// Parse command-line flags
	isGlobal := pflag.BoolP("global", "g", false, "scan global snippets directory (~)")
	isExactPath := pflag.BoolP("exact", "e", false, "use exact directory without auto-detecting ./"+autoDetectSubDir+"/")
	isShowVersion := pflag.BoolP("version", "v", false, "print version information")
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: snipfly [options] [directory]\n\nOptions:\n")
		pflag.PrintDefaults()
	}
	pflag.Parse()

	// Print version and exit if requested
	if *isShowVersion {
		fmt.Printf("snipfly version %s (commit: %s, built: %s)\n", Version, Commit, Date)
		return nil
	}

	// Determine scan directory
	args := pflag.Args()
	var scanDir string
	switch {
	case *isGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		scanDir = home
	case len(args) > 0:
		scanDir = args[0]
	default:
		scanDir = "."
	}

	// Auto-detect .snipfly/ subdirectory
	if !*isExactPath {
		subDir := filepath.Join(scanDir, autoDetectSubDir)
		if info, err := os.Stat(subDir); err == nil && info.IsDir() {
			scanDir = subDir
		}
	}

	// Scan for snippets
	snippets, err := snippet.ScanDir(scanDir)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}
	if len(snippets) == 0 {
		return fmt.Errorf("no snippets found in %s", scanDir)
	}

	// Create runner with nil callbacks (injected after program creation)
	r := runner.New(nil, nil)

	// Create TUI model and program
	model := tui.NewAppModel(snippets, r)
	p := tea.NewProgram(model)

	// Inject callbacks: Runner → Program.Send
	r.SetCallbacks(
		func(snippetPath, _ string) {
			p.Send(tui.OutputMsg{SnippetPath: snippetPath})
		},
		func(snippetPath string, exitCode int, err error) {
			p.Send(tui.ProcessExitedMsg{
				SnippetPath: snippetPath,
				ExitCode:    exitCode,
				Err:         err,
			})
		},
	)

	// Handle OS signals for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, platform.OsSignals()...)
	go func() {
		<-sigCh
		r.StopAll()
		p.Quit()
	}()

	// Run the TUI
	if _, err := p.Run(); err != nil {
		r.StopAll()
		return err
	}

	// Safety net: stop any remaining processes
	r.StopAll()
	return nil
}
