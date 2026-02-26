package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/txperl/GoSnippet/internal/runner"
	"github.com/txperl/GoSnippet/internal/snippet"
	"github.com/txperl/GoSnippet/internal/tui"

	tea "charm.land/bubbletea/v2"
)

func main() {
	dir := flag.String("l", "", "directory to scan for snippets")
	global := flag.Bool("g", false, "scan global snippets directory (~/.gosnippet)")
	flag.Parse()

	// Determine scan directory
	scanDir := "."
	switch {
	case *dir != "":
		scanDir = *dir
	case *global:
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		scanDir = home + "/.gosnippet"
	default:
		// If positional arg is provided, use it
		if flag.NArg() > 0 {
			scanDir = flag.Arg(0)
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
