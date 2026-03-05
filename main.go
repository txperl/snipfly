package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/txperl/snipfly/internal/runner"
	"github.com/txperl/snipfly/internal/snippet"
	"github.com/txperl/snipfly/internal/tui"

	tea "charm.land/bubbletea/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	autoDetectSubDir = ".snipfly"
)

func main() {
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
		fmt.Printf("snipfly version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Determine scan directory
	args := pflag.Args()
	var scanDir string
	switch {
	case *isGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}
	if len(snippets) == 0 {
		fmt.Fprintf(os.Stderr, "No snippets found in %s\n", scanDir)
		os.Exit(1)
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
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigCh
		r.StopAll()
		p.Quit()
	}()

	// Run the TUI
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		r.StopAll()
		os.Exit(1)
	}

	// Safety net: stop any remaining processes
	r.StopAll()
}
