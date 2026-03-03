package progressui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/log"
	"github.com/mattn/go-isatty"
)

type Mode string

type Theme string

const (
	ModeAuto Mode = "auto"
	ModeOn   Mode = "on"
	ModeOff  Mode = "off"
)

const (
	ThemeCalm Theme = "calm"
)

const (
	defaultSpinnerTick       = 100 * time.Millisecond
	defaultBarWidth          = 30
	defaultBarMinWidth       = 20
	defaultBarMaxWidth       = 80
	defaultBarSidePadding    = 6
	defaultVisibleFileStatus = 8
	envProgressDebug         = "HYPERLOCALISE_PROGRESS_DEBUG"
	envProgressDebugFilePath = "HYPERLOCALISE_PROGRESS_DEBUG_FILE"
	defaultDebugLogPath      = ".hyperlocalise/logs/run.log"
)

type Options struct {
	Label          string
	Tick           time.Duration
	Frames         []string
	IsTTYFn        func(io.Writer) bool
	OnInterrupt    func()
	Theme          Theme
	EnableDebugLog bool
	DebugLogPath   string
}

type Renderer struct {
	w           io.Writer
	mode        Mode
	interactive bool

	mu           sync.Mutex
	program      *tea.Program
	doneCh       chan struct{}
	closed       bool
	completed    bool
	total        int
	succeeded    int
	failed       int
	lastPlainLog time.Time

	logger  *log.Logger
	logFile *os.File

	interruptOnce sync.Once
	onInterrupt   func()
}

type phaseMsg struct {
	text string
}

type planMsg struct {
	total int
}

type taskDoneMsg struct {
	succeeded int
	failed    int
	total     int
}

type taskStartedMsg struct {
	targetPath string
	entryKey   string
}

type taskStatusMsg struct {
	targetPath    string
	entryKey      string
	taskSucceeded bool
	failureReason string
}

type completeMsg struct{}

type model struct {
	label     string
	phase     string
	modeLabel string
	total     int
	succeeded int
	failed    int
	done      bool

	spin spinner.Model
	bar  progress.Model

	styles dashboardStyles

	files      map[string]fileStatus
	fileOrder  []string
	maxVisible int
}

type fileStatus struct {
	targetPath string
	lastEntry  string
	lastReason string
	processing int
	succeeded  int
	failed     int
}

type dashboardStyles struct {
	container lipgloss.Style
	header    lipgloss.Style
	progress  lipgloss.Style
	summary   lipgloss.Style
	ok        lipgloss.Style
	fail      lipgloss.Style
	meta      lipgloss.Style
	files     lipgloss.Style
	fileLine  lipgloss.Style
	fileDone  lipgloss.Style
	fileBusy  lipgloss.Style
	fileError lipgloss.Style
}

func ParseMode(raw string) (Mode, error) {
	mode := Mode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case ModeAuto, ModeOn, ModeOff:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid --progress value %q: must be one of auto|on|off", raw)
	}
}

func IsEnabled(mode Mode, w io.Writer, isTTYFn func(io.Writer) bool) bool {
	switch mode {
	case ModeOff:
		return false
	case ModeOn:
		return true
	case ModeAuto:
		return detectTTY(w, isTTYFn)
	default:
		return false
	}
}

func New(w io.Writer, mode Mode, options Options) *Renderer {
	tick := options.Tick
	if tick <= 0 {
		tick = defaultSpinnerTick
	}

	label := strings.TrimSpace(options.Label)
	if label == "" {
		label = "Working"
	}

	r := &Renderer{
		w:           w,
		mode:        mode,
		interactive: detectTTY(w, options.IsTTYFn),
		onInterrupt: options.OnInterrupt,
	}

	enabled, logPath := resolveDebugLogConfig(options)
	if enabled {
		var err error
		r.logger, r.logFile, err = newDebugLogger(logPath)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "hyperlocalise: progress debug logging disabled: %v\n", err)
		} else {
			r.debug("renderer started", "interactive", r.interactive, "mode", r.mode, "log_file", logPath)
		}
	}

	if !r.interactive {
		return r
	}

	initial := newModel(label, mode, tick, options)
	program := tea.NewProgram(
		initial,
		tea.WithOutput(w),
		tea.WithInput(os.Stdin),
		tea.WithoutSignalHandler(),
		tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			switch typed := msg.(type) {
			case tea.KeyPressMsg:
				if typed.String() == "ctrl+c" {
					r.triggerInterrupt()
					return tea.QuitMsg{}
				}
			case tea.InterruptMsg:
				r.triggerInterrupt()
				return tea.QuitMsg{}
			}

			return msg
		}),
	)
	r.program = program
	r.doneCh = make(chan struct{})

	go func() {
		_, _ = program.Run()
		close(r.doneCh)
	}()

	return r
}

func (r *Renderer) Phase(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed || strings.TrimSpace(message) == "" {
		return
	}

	r.debug("phase", "text", message)

	if r.interactive {
		r.program.Send(phaseMsg{text: message})
		return
	}

	r.logPlain(fmt.Sprintf("progress phase=%s", message), false)
}

func (r *Renderer) Plan(total int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}

	r.total = total
	r.debug("plan", "total", total)
	if r.interactive {
		r.program.Send(planMsg{total: total})
		return
	}

	r.logPlain(fmt.Sprintf("progress executable_total=%d", total), true)
}

func (r *Renderer) TaskStarted(targetPath, entryKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed || strings.TrimSpace(targetPath) == "" {
		return
	}

	r.debug("task start", "target_path", targetPath, "entry_key", entryKey)

	if r.interactive {
		r.program.Send(taskStartedMsg{targetPath: targetPath, entryKey: entryKey})
	}
}

func (r *Renderer) TaskStatus(targetPath, entryKey string, taskSucceeded bool, failureReason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed || strings.TrimSpace(targetPath) == "" {
		return
	}

	r.debug("task status", "target_path", targetPath, "entry_key", entryKey, "task_succeeded", taskSucceeded, "failure_reason", failureReason)

	if r.interactive {
		r.program.Send(taskStatusMsg{
			targetPath:    targetPath,
			entryKey:      entryKey,
			taskSucceeded: taskSucceeded,
			failureReason: failureReason,
		})
	}
}

func (r *Renderer) TaskDone(succeeded, failed, total int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}

	r.succeeded = succeeded
	r.failed = failed
	if total > 0 {
		r.total = total
	}

	r.debug("task done", "succeeded", succeeded, "failed", failed, "total", total)

	if r.interactive {
		r.program.Send(taskDoneMsg{failed: failed, succeeded: succeeded, total: total})
		return
	}

	completed := succeeded + failed
	line := fmt.Sprintf("progress completed=%d/%d succeeded=%d failed=%d", completed, r.total, succeeded, failed)
	r.logPlain(line, completed == r.total)
}

func (r *Renderer) Complete() {
	r.mu.Lock()
	if r.closed || r.completed {
		r.mu.Unlock()
		return
	}
	r.completed = true
	interactive := r.interactive
	program := r.program
	doneCh := r.doneCh
	succeeded := r.succeeded
	failed := r.failed
	r.mu.Unlock()

	r.debug("complete", "succeeded", succeeded, "failed", failed)

	if interactive {
		program.Send(completeMsg{})
		if doneCh != nil {
			<-doneCh
		}
		return
	}

	_, _ = fmt.Fprintf(r.w, "progress done succeeded=%d failed=%d\n", succeeded, failed)
}

func (r *Renderer) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	interactive := r.interactive
	program := r.program
	doneCh := r.doneCh
	logFile := r.logFile
	r.mu.Unlock()

	r.debug("close")

	if interactive {
		program.Quit()
		if doneCh != nil {
			<-doneCh
		}
	}

	if logFile != nil {
		_ = logFile.Close()
	}
}

func (r *Renderer) logPlain(line string, force bool) {
	if r.mode == ModeAuto {
		return
	}

	now := time.Now()
	if !force && now.Sub(r.lastPlainLog) < time.Second {
		return
	}
	r.lastPlainLog = now
	_, _ = fmt.Fprintln(r.w, line)
}

func detectTTY(w io.Writer, isTTYFn func(io.Writer) bool) bool {
	if isTTYFn != nil {
		return isTTYFn(w)
	}

	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func newModel(label string, mode Mode, tick time.Duration, options Options) model {
	frames := options.Frames
	if len(frames) == 0 {
		frames = spinner.MiniDot.Frames
	}

	styles := newDashboardStyles(options.Theme)
	spin := spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: frames,
		FPS:    tick,
	}))
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	bar := progress.New(progress.WithWidth(defaultBarWidth), progress.WithDefaultBlend())

	return model{
		label:      label,
		phase:      "Working...",
		modeLabel:  string(mode),
		spin:       spin,
		bar:        bar,
		styles:     styles,
		files:      map[string]fileStatus{},
		maxVisible: defaultVisibleFileStatus,
	}
}

func newDashboardStyles(theme Theme) dashboardStyles {
	effectiveTheme := theme
	if effectiveTheme == "" {
		effectiveTheme = ThemeCalm
	}

	container := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	progressLine := lipgloss.NewStyle().Foreground(lipgloss.Color("249"))
	summary := lipgloss.NewStyle().Foreground(lipgloss.Color("247"))
	meta := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))

	okColor := lipgloss.Color("78")
	if effectiveTheme != ThemeCalm {
		okColor = lipgloss.Color("78")
	}
	ok := lipgloss.NewStyle().Foreground(okColor).Bold(true)
	fail := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)

	return dashboardStyles{
		container: container,
		header:    header,
		progress:  progressLine,
		summary:   summary,
		ok:        ok,
		fail:      fail,
		meta:      meta,
		files:     lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Bold(true),
		fileLine:  lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		fileDone:  lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Bold(true),
		fileBusy:  lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Bold(true),
		fileError: lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
	}
}

func (m model) Init() tea.Cmd {
	return m.spin.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.done {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		bar, cmd := m.bar.Update(msg)
		m.bar = bar
		return m, cmd
	case tea.WindowSizeMsg:
		m.bar.SetWidth(clampBarWidth(msg.Width - defaultBarSidePadding))
		return m, nil
	case phaseMsg:
		m.phase = msg.text
		return m, nil
	case planMsg:
		m.total = msg.total
		return m, m.setProgressCmd()
	case taskStartedMsg:
		m.recordTaskStarted(msg.targetPath, msg.entryKey)
		return m, nil
	case taskStatusMsg:
		m.recordTaskFinished(msg.targetPath, msg.entryKey, msg.taskSucceeded, msg.failureReason)
		return m, nil
	case taskDoneMsg:
		m.succeeded = msg.succeeded
		m.failed = msg.failed
		if msg.total > 0 {
			m.total = msg.total
		}
		return m, m.setProgressCmd()
	case completeMsg:
		m.done = true
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m model) View() tea.View {
	if m.done {
		return tea.NewView("")
	}

	phase := m.phase
	if strings.TrimSpace(phase) == "" {
		phase = "Working..."
	}

	headerLine := m.styles.header.Render(fmt.Sprintf("%s %s", m.spin.View(), phase))
	completed := m.succeeded + m.failed

	progressLine := ""
	statusText := "mode=indeterminate"
	if m.total > 0 {
		progressLine = m.styles.progress.Render(fmt.Sprintf("%s %d/%d", m.bar.View(), completed, m.total))
		statusText = "mode=determinate"
	} else {
		progressLine = m.styles.progress.Render("estimating workload...")
	}

	summaryLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.styles.summary.Render(m.label+":"),
		" ",
		m.styles.ok.Render(fmt.Sprintf("ok=%d", m.succeeded)),
		"  ",
		m.styles.fail.Render(fmt.Sprintf("fail=%d", m.failed)),
		"  ",
		m.styles.meta.Render(statusText+" progress="+m.modeLabel),
	)

	sections := []string{headerLine, progressLine, summaryLine}
	if filesBlock := m.fileStatusView(); filesBlock != "" {
		sections = append(sections, filesBlock)
	}

	return tea.NewView(m.styles.container.Render(lipgloss.JoinVertical(lipgloss.Left, sections...)))
}

func (m *model) setProgressCmd() tea.Cmd {
	if m.total <= 0 {
		return nil
	}
	completed := m.succeeded + m.failed
	if completed < 0 {
		completed = 0
	}
	if completed > m.total {
		completed = m.total
	}
	return m.bar.SetPercent(float64(completed) / float64(m.total))
}

func clampBarWidth(w int) int {
	if w < defaultBarMinWidth {
		return defaultBarMinWidth
	}
	if w > defaultBarMaxWidth {
		return defaultBarMaxWidth
	}
	return w
}

func (m *model) recordTaskStarted(targetPath, entryKey string) {
	path := strings.TrimSpace(targetPath)
	if path == "" {
		return
	}

	status, ok := m.files[path]
	if !ok {
		status = fileStatus{targetPath: path}
	}
	status.processing++
	status.lastEntry = entryKey
	status.lastReason = ""
	m.files[path] = status
	m.touchFile(path)
}

func (m *model) recordTaskFinished(targetPath, entryKey string, taskSucceeded bool, failureReason string) {
	path := strings.TrimSpace(targetPath)
	if path == "" {
		return
	}

	status, ok := m.files[path]
	if !ok {
		status = fileStatus{targetPath: path}
	}
	if status.processing > 0 {
		status.processing--
	}
	status.lastEntry = entryKey
	if taskSucceeded {
		status.succeeded++
		status.lastReason = ""
	} else {
		status.failed++
		status.lastReason = failureReason
	}
	m.files[path] = status
	m.touchFile(path)
}

func (m *model) touchFile(path string) {
	next := make([]string, 0, len(m.fileOrder)+1)
	next = append(next, path)
	for _, existing := range m.fileOrder {
		if existing == path {
			continue
		}
		next = append(next, existing)
	}
	m.fileOrder = next
}

func (m model) fileStatusView() string {
	if len(m.fileOrder) == 0 {
		return ""
	}

	limit := m.maxVisible
	if limit <= 0 {
		limit = defaultVisibleFileStatus
	}
	sorted := m.sortedFilePaths()
	if limit > len(sorted) {
		limit = len(sorted)
	}

	lines := make([]string, 0, limit+1)
	lines = append(lines, m.styles.files.Render("Files"))
	for _, path := range sorted[:limit] {
		status := m.files[path]
		state, _ := fileState(status)
		fileName := statusLabel(path)
		switch state {
		case "processing":
			fileName = m.styles.fileBusy.Render(fileName)
		case "failed":
			fileName = m.styles.fileError.Render(fileName)
		default:
			fileName = m.styles.fileDone.Render(fileName)
		}

		row := fmt.Sprintf("- %s [%s] ok=%d fail=%d", fileName, state, status.succeeded, status.failed)
		if status.lastEntry != "" {
			row += " key=" + status.lastEntry
		}
		if status.lastReason != "" {
			row += " reason=" + status.lastReason
		}
		lines = append(lines, m.styles.fileLine.Render(row))
	}

	if len(sorted) > limit {
		lines = append(lines, m.styles.meta.Render(fmt.Sprintf("... %d more files", len(sorted)-limit)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) sortedFilePaths() []string {
	paths := append([]string(nil), m.fileOrder...)
	if len(paths) <= 1 {
		return paths
	}

	rank := make(map[string]int, len(m.fileOrder))
	for i, path := range m.fileOrder {
		rank[path] = i
	}

	sort.SliceStable(paths, func(i, j int) bool {
		left := m.files[paths[i]]
		right := m.files[paths[j]]
		_, leftPriority := fileState(left)
		_, rightPriority := fileState(right)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return rank[paths[i]] < rank[paths[j]]
	})

	return paths
}

func fileState(status fileStatus) (state string, priority int) {
	switch {
	case status.processing > 0:
		return "processing", 0
	case status.failed > 0:
		return "failed", 1
	default:
		return "done", 2
	}
}

func statusLabel(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "(unknown)"
	}
	base := filepath.Base(trimmed)
	if base == "." || base == string(os.PathSeparator) {
		return trimmed
	}
	return base
}

func resolveDebugLogConfig(options Options) (bool, string) {
	enabled := options.EnableDebugLog || parseBoolEnv(os.Getenv(envProgressDebug))
	if !enabled {
		return false, ""
	}

	path := strings.TrimSpace(options.DebugLogPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv(envProgressDebugFilePath))
	}
	if path == "" {
		path = defaultDebugLogPath
	}

	return true, path
}

func newDebugLogger(path string) (*log.Logger, *os.File, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil, fmt.Errorf("empty debug log path")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, fmt.Errorf("create debug log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open debug log file %q: %w", path, err)
	}

	logger := log.NewWithOptions(file, log.Options{
		Level:           log.DebugLevel,
		ReportTimestamp: true,
		Formatter:       log.LogfmtFormatter,
		Prefix:          "progressui",
	})

	return logger, file, nil
}

func parseBoolEnv(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return strings.EqualFold(strings.TrimSpace(raw), "on")
	}

	return parsed
}

func (r *Renderer) debug(message string, keyvals ...interface{}) {
	if r.logger == nil {
		return
	}
	r.logger.Debug(message, keyvals...)
}

func (r *Renderer) triggerInterrupt() {
	r.interruptOnce.Do(func() {
		if r.onInterrupt != nil {
			r.onInterrupt()
		}
	})
}
