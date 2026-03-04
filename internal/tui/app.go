package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/txperl/GoSnippet/internal/runner"
	"github.com/txperl/GoSnippet/internal/snippet"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// FocusPanel indicates which panel currently has keyboard focus.
type FocusPanel int

const (
	FocusList FocusPanel = iota
	FocusOutput
)

// AppModel is the root Bubble Tea model for GoSnippet.
type AppModel struct {
	snippets []snippet.Snippet
	runner   *runner.Runner

	list   ListModel
	output OutputModel

	termWidth  int
	termHeight int

	listWidth     int
	outputWidth   int
	contentHeight int

	focus        FocusPanel
	selectedPath string

	showConfirm    bool
	outputPending  bool
	throttleActive bool

	execResult map[string]string // last exit info for interactive snippets
}

// NewAppModel creates the root TUI model.
func NewAppModel(snippets []snippet.Snippet, r *runner.Runner) AppModel {
	m := AppModel{
		snippets:   snippets,
		runner:     r,
		output:     NewOutputModel(),
		execResult: make(map[string]string),
	}
	m.list = NewListModel(m.snippets, r.GetState)
	m.list.focused = true
	// Select the first snippet if available
	if sel := m.list.SelectedSnippet(); sel != nil {
		m.selectedPath = sel.FilePath
	}
	return m
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case OutputMsg:
		return m.handleOutput(msg)
	case OutputThrottleTickMsg:
		return m.handleThrottleTick()
	case ProcessExitedMsg:
		return m.handleProcessExited(msg)
	case ExecFinishedMsg:
		return m.handleExecFinished(msg)
	}

	// Pass to output panel
	var cmd tea.Cmd
	m.output, cmd = m.output.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m AppModel) View() tea.View {
	if m.termWidth == 0 || m.termHeight == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	var content string
	if m.showConfirm {
		content = m.viewWithConfirm()
	} else {
		content = m.viewNormal()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m *AppModel) viewNormal() string {
	body := m.renderBody()
	statusBar := m.renderStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
}

func (m *AppModel) renderHeader() string {
	title := StyleHeader.Render("GoSnippet")
	quit := StyleHeader.Render("[Q]uit")
	spacer := strings.Repeat(" ", max(0, m.termWidth-lipgloss.Width(title)-lipgloss.Width(quit)))
	return title + spacer + quit
}

func (m *AppModel) renderBody() string {
	listView := m.list.View()
	outputView := m.renderOutput()
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, outputView)
}

func (m *AppModel) renderOutput() string {
	// Render title bar
	var title string
	if m.focus == FocusOutput {
		title = StylePanelTitle.Width(m.outputWidth).Render("◆ Output")
	} else {
		title = StylePanelTitleBlurred.Width(m.outputWidth).Render("  Output")
	}

	sel := m.list.SelectedSnippet()
	mh := metadataHeight(sel)
	vpHeight := m.contentHeight - mh - 1 // -1 for title bar
	if vpHeight < 1 {
		vpHeight = 1
	}

	vpStyle := lipgloss.NewStyle().
		Width(m.outputWidth).
		Height(vpHeight)

	if mh == 0 {
		// No metadata — full height viewport
		return lipgloss.JoinVertical(lipgloss.Left, title,
			lipgloss.NewStyle().
				Width(m.outputWidth).
				Height(m.contentHeight-1).
				Render(m.output.View()))
	}

	meta := renderMetadata(sel, m.outputWidth)
	return lipgloss.JoinVertical(lipgloss.Left, title, meta, vpStyle.Render(m.output.View()))
}

// --- Resize ---

func (m AppModel) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.termWidth = msg.Width
	m.termHeight = msg.Height
	m.recalcLayout()
	return m, nil
}

func (m *AppModel) recalcLayout() {
	m.contentHeight = m.termHeight - 1 // status bar
	if m.contentHeight < 1 {
		m.contentHeight = 1
	}

	m.listWidth = m.termWidth * 30 / 100
	if m.listWidth < 20 {
		m.listWidth = 20
	}
	if m.listWidth > 40 {
		m.listWidth = 40
	}

	m.outputWidth = m.termWidth - m.listWidth - 1 // -1 for border
	if m.outputWidth < 1 {
		m.outputWidth = 1
	}

	m.list.SetSize(m.listWidth, m.contentHeight)
	m.resizeOutputViewport()
}

// resizeOutputViewport recalculates the viewport height based on the
// current snippet's metadata height and updates the output panel size.
func (m *AppModel) resizeOutputViewport() {
	sel := m.list.SelectedSnippet()
	mh := metadataHeight(sel)
	vpHeight := m.contentHeight - mh - 1 // -1 for title bar
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.output.SetSize(m.outputWidth, vpHeight)
}

// --- Key handling ---

func (m AppModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.showConfirm {
		return m.handleConfirmKey(msg)
	}

	key := msg.String()
	switch key {
	case "q", "Q":
		if m.runner.HasRunning() {
			m.showConfirm = true
			return m, nil
		}
		return m, tea.Quit

	case "tab":
		if m.focus == FocusList {
			m.focus = FocusOutput
		} else {
			m.focus = FocusList
		}
		m.list.focused = m.focus == FocusList
		return m, nil

	case "space":
		if m.focus == FocusList {
			sel := m.list.SelectedSnippet()
			if sel != nil && m.runner.GetState(sel.FilePath) == snippet.StateRunning {
				return m.handleStop()
			}
			return m.handleSpace()
		}

	case "r", "R":
		if m.focus == FocusList {
			return m.handleRestart()
		}
	}

	// Delegate to focused panel
	if m.focus == FocusList {
		return m.handleListKey(key)
	}
	return m.handleOutputKey(msg)
}

func (m AppModel) handleListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		m.list.MoveUp()
		m.updateSelectedPath()
		m.resizeOutputViewport()
		m.refreshOutputContent()
	case "down", "j":
		m.list.MoveDown()
		m.updateSelectedPath()
		m.resizeOutputViewport()
		m.refreshOutputContent()
	}
	return m, nil
}

func (m AppModel) handleOutputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.output, cmd = m.output.Update(msg)
	return m, cmd
}

func (m *AppModel) updateSelectedPath() {
	if sel := m.list.SelectedSnippet(); sel != nil {
		m.selectedPath = sel.FilePath
	}
}

// --- Actions ---

func (m AppModel) handleSpace() (tea.Model, tea.Cmd) {
	sel := m.list.SelectedSnippet()
	if sel == nil {
		return m, nil
	}
	if sel.Error != "" {
		m.refreshOutputContent()
		return m, nil
	}

	// Interactive type: exec full-screen via tea.ExecProcess
	if sel.Type == snippet.TypeInteractive {
		args := append(sel.InterpreterArgs, sel.FilePath)
		cmd := exec.Command(sel.Interpreter, args...)
		cmd.Dir = sel.Dir
		cmd.Env = append(os.Environ(), sel.Env...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		path := sel.FilePath
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return ExecFinishedMsg{SnippetPath: path, Err: err}
		})
	}

	state := m.runner.GetState(sel.FilePath)
	if state == snippet.StateRunning {
		return m, nil
	}
	if err := m.runner.Start(*sel); err != nil {
		m.output.SetContent(fmt.Sprintf("Error: %s\n\nThis snippet cannot be started.", err))
		return m, nil
	}
	m.refreshOutputContent()
	return m, nil
}

func (m AppModel) handleStop() (tea.Model, tea.Cmd) {
	sel := m.list.SelectedSnippet()
	if sel == nil {
		return m, nil
	}
	if sel.Type == snippet.TypeInteractive {
		return m, nil
	}
	m.runner.Stop(*sel)
	m.refreshOutputContent()
	return m, nil
}

func (m AppModel) handleRestart() (tea.Model, tea.Cmd) {
	sel := m.list.SelectedSnippet()
	if sel == nil {
		return m, nil
	}
	if sel.Type == snippet.TypeInteractive {
		return m, nil
	}
	s := *sel
	r := m.runner
	// Restart blocks (Stop+Wait+Start), so run in a goroutine
	return m, func() tea.Msg {
		_ = r.Restart(s)
		return OutputMsg{SnippetPath: s.FilePath}
	}
}

// --- Output throttling ---

func (m AppModel) handleOutput(_ OutputMsg) (tea.Model, tea.Cmd) {
	m.outputPending = true
	if !m.throttleActive {
		m.throttleActive = true
		return m, throttleTickCmd()
	}
	return m, nil
}

func (m AppModel) handleThrottleTick() (tea.Model, tea.Cmd) {
	m.throttleActive = false
	if m.outputPending {
		m.refreshOutputContent()
		m.outputPending = false
	}
	return m, nil
}

func (m AppModel) handleProcessExited(msg ProcessExitedMsg) (tea.Model, tea.Cmd) {
	_ = msg
	m.refreshOutputContent()
	return m, nil
}

func (m AppModel) handleExecFinished(msg ExecFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.execResult[msg.SnippetPath] = fmt.Sprintf("--- Interactive process exited with error: %s ---", msg.Err)
	} else {
		m.execResult[msg.SnippetPath] = "--- Interactive process exited (code: 0) ---"
	}
	m.refreshOutputContent()
	return m, nil
}

// refreshOutputContent reads the buffer for the currently selected snippet
// and updates the output panel content.
func (m *AppModel) refreshOutputContent() {
	if m.selectedPath == "" {
		m.output.SetContent("Select a snippet and press Space to run it.")
		return
	}

	sel := m.list.SelectedSnippet()

	// Check if the selected snippet has a build-time error
	if sel != nil && sel.Error != "" {
		m.output.SetContent(fmt.Sprintf("Error: %s\n\nThis snippet cannot be run.", sel.Error))
		return
	}

	// Interactive type: show last exit result or prompt
	if sel != nil && sel.Type == snippet.TypeInteractive {
		if result, ok := m.execResult[m.selectedPath]; ok {
			m.output.SetContent(result)
		} else {
			m.output.SetContent("Press Space to launch (full-screen mode).")
		}
		return
	}

	buf := m.runner.GetBuffer(m.selectedPath)
	if buf == nil {
		state := m.runner.GetState(m.selectedPath)
		m.output.SetContent(fmt.Sprintf("State: %s\n\nPress Space to run.", state))
		return
	}

	lines := buf.Lines()
	content := strings.Join(lines, "\n")

	// Append exit info for terminal states
	state := m.runner.GetState(m.selectedPath)
	if exitSuffix := m.exitInfoSuffix(state); exitSuffix != "" {
		if content != "" {
			content += "\n"
		}
		content += exitSuffix
	}

	if content == "" {
		m.output.SetContent(fmt.Sprintf("State: %s\n\n(no output yet)", state))
		return
	}

	m.output.SetContent(content)
}

// exitInfoSuffix returns the exit summary line for terminal process states.
func (m *AppModel) exitInfoSuffix(state snippet.ProcessState) string {
	switch state {
	case snippet.StateDone, snippet.StateExited:
		return "\n--- Process exited (code: 0) ---"
	case snippet.StateFailed, snippet.StateCrashed:
		proc := m.runner.GetProcess(m.selectedPath)
		code := -1
		if proc != nil {
			code = proc.ExitCode()
		}
		suffix := fmt.Sprintf("\n--- Process exited (code: %d) ---", code)
		if state == snippet.StateCrashed {
			suffix += "\nPress Space to re-run."
		}
		return suffix
	case snippet.StateStopped:
		return "\n--- Process stopped ---"
	default:
		return ""
	}
}
