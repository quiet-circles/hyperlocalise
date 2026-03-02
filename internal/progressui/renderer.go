package progressui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-isatty"
)

type Mode string

const (
	ModeAuto Mode = "auto"
	ModeOn   Mode = "on"
	ModeOff  Mode = "off"
)

const (
	defaultSpinnerTick = 100 * time.Millisecond
	defaultBarWidth    = 30
)

type Options struct {
	Label   string
	Tick    time.Duration
	Frames  []string
	IsTTYFn func(io.Writer) bool
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

type spinnerTickMsg struct{}

type model struct {
	label      string
	phase      string
	frames     []string
	tick       time.Duration
	frameIndex int
	total      int
	succeeded  int
	failed     int
	done       bool
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

	frames := options.Frames
	if len(frames) == 0 {
		frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
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

	if !r.interactive {
		return r
	}

	initial := model{
		label:  label,
		phase:  "Working...",
		frames: frames,
		tick:   tick,
	}

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
	r.mu.Unlock()

	if interactive {
		program.Quit()
		if doneCh != nil {
			<-doneCh
		}
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

func (m model) Init() tea.Cmd {
	return tickCmd(m.tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerTickMsg:
		if m.done {
			return m, nil
		}
		m.frameIndex = (m.frameIndex + 1) % len(m.frames)
		return m, tickCmd(m.tick)
	case phaseMsg:
		m.phase = msg.text
		return m, nil
	case planMsg:
		m.total = msg.total
		return m, nil
	case taskDoneMsg:
		m.succeeded = msg.succeeded
		m.failed = msg.failed
		if msg.total > 0 {
			m.total = msg.total
		}
		return m, nil
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

	frame := m.frames[m.frameIndex]
	completed := m.succeeded + m.failed

	if m.total <= 0 {
		return tea.NewView(fmt.Sprintf("%s %s ok=%d fail=%d", frame, phase, m.succeeded, m.failed))
	}

	filled := 0
	if completed > 0 {
		filled = int(float64(completed) / float64(m.total) * defaultBarWidth)
		if filled > defaultBarWidth {
			filled = defaultBarWidth
		}
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("=", filled) + strings.Repeat("-", defaultBarWidth-filled)

	return tea.NewView(fmt.Sprintf("%s %s\n[%s] %d/%d ok=%d fail=%d", frame, phase, bar, completed, m.total, m.succeeded, m.failed))
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}
