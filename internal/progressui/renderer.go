package progressui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	envProgressDebug         = "HYPERLOCALISE_PROGRESS_DEBUG"
	envProgressDebugFilePath = "HYPERLOCALISE_PROGRESS_DEBUG_FILE"
	defaultDebugLogPath      = ".hyperlocalise/logs/run.log"
)

type Options struct {
	Label          string
	Tick           time.Duration
	Frames         []string
	IsTTYFn        func(io.Writer) bool
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

type completeMsg struct{}

type model struct {
	label     string
	phase     string
	modeLabel string
	total     int
	succeeded int
	failed    int
	done      bool

	spinner spinner.Model
	bar     progress.Model
	styles  dashboardStyles
}

type dashboardStyles struct {
	container lipgloss.Style
	header    lipgloss.Style
	progress  lipgloss.Style
	summary   lipgloss.Style
	ok        lipgloss.Style
	fail      lipgloss.Style
	meta      lipgloss.Style
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
	}

	enabled, logPath := resolveDebugLogConfig(options)
	if enabled {
		r.logger, r.logFile = newDebugLogger(logPath)
		r.debug("renderer started", "interactive", r.interactive, "mode", r.mode, "log_file", logPath)
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

	spin := spinner.New(spinner.WithSpinner(spinner.Spinner{Frames: frames, FPS: tick}))
	styles := newDashboardStyles(options.Theme)
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	bar := progress.New(progress.WithWidth(defaultBarWidth), progress.WithoutPercentage())
	bar.Full = '█'
	bar.Empty = '░'
	bar.FullColor = "#7AA2F7"
	bar.EmptyColor = "#3A3A45"

	return model{
		label:     label,
		phase:     "Working...",
		modeLabel: string(mode),
		spinner:   spin,
		bar:       bar,
		styles:    styles,
	}
}

func newDashboardStyles(theme Theme) dashboardStyles {
	if theme == "" {
		theme = ThemeCalm
	}

	container := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	progressLine := lipgloss.NewStyle().Foreground(lipgloss.Color("249"))
	summary := lipgloss.NewStyle().Foreground(lipgloss.Color("247"))
	ok := lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Bold(true)
	fail := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	meta := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))

	return dashboardStyles{
		container: container,
		header:    header,
		progress:  progressLine,
		summary:   summary,
		ok:        ok,
		fail:      fail,
		meta:      meta,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		next, cmd := m.bar.Update(msg)
		bar, ok := next.(progress.Model)
		if ok {
			m.bar = bar
		}
		return m, cmd
	case phaseMsg:
		m.phase = msg.text
		return m, nil
	case planMsg:
		m.total = msg.total
		return m, m.setProgressCmd()
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

func (m model) View() string {
	if m.done {
		return ""
	}

	phase := m.phase
	if strings.TrimSpace(phase) == "" {
		phase = "Working..."
	}

	headerLine := m.styles.header.Render(fmt.Sprintf("%s %s", m.spinner.View(), phase))
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

	return m.styles.container.Render(lipgloss.JoinVertical(lipgloss.Left, headerLine, progressLine, summaryLine))
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

	percent := float64(completed) / float64(m.total)
	return m.bar.SetPercent(percent)
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

func newDebugLogger(path string) (*log.Logger, *os.File) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil
	}

	logger := log.NewWithOptions(file, log.Options{
		Level:           log.DebugLevel,
		ReportTimestamp: true,
		Formatter:       log.LogfmtFormatter,
		Prefix:          "progressui",
	})

	return logger, file
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
