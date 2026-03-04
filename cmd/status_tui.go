package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-isatty"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

type statusSummary struct {
	Total        int
	Translated   int
	NeedsReview  int
	Untranslated int
	ByLocale     []localeSummary
}

type localeSummary struct {
	Locale       string
	Total        int
	Translated   int
	NeedsReview  int
	Untranslated int
}

type statusSortMode int

const (
	sortByLocale statusSortMode = iota
	sortByUntranslated
	sortByTranslated
)

type statusKeyMap struct {
	Sort       key.Binding
	Reverse    key.Binding
	ToggleHelp key.Binding
	Quit       key.Binding
}

func defaultStatusKeyMap() statusKeyMap {
	return statusKeyMap{
		Sort: key.NewBinding(
			key.WithKeys("s", "tab"),
			key.WithHelp("s/tab", "sort"),
		),
		Reverse: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reverse"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k statusKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Sort, k.Reverse, k.ToggleHelp, k.Quit}
}

func (k statusKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Sort, k.Reverse, k.ToggleHelp, k.Quit}}
}

type statusDashboardModel struct {
	summary  statusSummary
	group    string
	bucket   string
	sortMode statusSortMode
	reverse  bool

	keys statusKeyMap
	help help.Model
	tbl  table.Model

	width  int
	height int

	titleStyle         lipgloss.Style
	metaStyle          lipgloss.Style
	overallStyle       lipgloss.Style
	translatedStyle    lipgloss.Style
	needsReviewStyle   lipgloss.Style
	untranslatedStyle  lipgloss.Style
	selectedLocaleInfo lipgloss.Style
}

func runStatusDashboard(w io.Writer, entries []storage.Entry, locales []string, group, bucket string) error {
	if !isTTYWriter(w) {
		return fmt.Errorf("--tty requires a TTY output")
	}

	summary := buildStatusSummary(entries, locales)
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("39"))
	styles.Selected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))

	cols := []table.Column{
		{Title: "Locale", Width: 12},
		{Title: "Total", Width: 8},
		{Title: "Translated", Width: 12},
		{Title: "Needs Review", Width: 14},
		{Title: "Untranslated", Width: 13},
		{Title: "Completion", Width: 11},
	}

	m := statusDashboardModel{
		summary: summary,
		group:   group,
		bucket:  bucket,
		keys:    defaultStatusKeyMap(),
		help:    help.New(),
		tbl: table.New(
			table.WithColumns(cols),
			table.WithHeight(15),
			table.WithFocused(true),
			table.WithStyles(styles),
		),
		titleStyle:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45")),
		metaStyle:          lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		overallStyle:       lipgloss.NewStyle().Bold(true),
		translatedStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		needsReviewStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		untranslatedStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		selectedLocaleInfo: lipgloss.NewStyle().Foreground(lipgloss.Color("111")),
	}
	m.sortRows()

	p := tea.NewProgram(
		m,
		tea.WithOutput(w),
		tea.WithInput(os.Stdin),
	)
	_, err := p.Run()
	return err
}

func (m statusDashboardModel) Init() tea.Cmd { return nil }

func (m statusDashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		tblWidth := max(40, msg.Width-2)
		tblHeight := max(8, msg.Height-9)
		m.tbl.SetWidth(tblWidth)
		m.tbl.SetHeight(tblHeight)
		m.help.SetWidth(msg.Width)
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Sort):
			m.sortMode = (m.sortMode + 1) % 3
			m.sortRows()
			return m, nil
		case key.Matches(msg, m.keys.Reverse):
			m.reverse = !m.reverse
			m.sortRows()
			return m, nil
		case key.Matches(msg, m.keys.ToggleHelp):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m statusDashboardModel) View() tea.View {
	title := m.titleStyle.Render("hyperlocalise status dashboard")
	scope := m.metaStyle.Render(fmt.Sprintf("group=%s  bucket=%s  sort=%s  reverse=%t", emptyDash(m.group), emptyDash(m.bucket), m.sortModeLabel(), m.reverse))
	overall := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.overallStyle.Render("overall "),
		fmt.Sprintf("total=%d  ", m.summary.Total),
		m.translatedStyle.Render(fmt.Sprintf("translated=%d", m.summary.Translated)),
		"  ",
		m.needsReviewStyle.Render(fmt.Sprintf("needs_review=%d", m.summary.NeedsReview)),
		"  ",
		m.untranslatedStyle.Render(fmt.Sprintf("untranslated=%d", m.summary.Untranslated)),
	)

	selectedInfo := ""
	if row := m.tbl.SelectedRow(); len(row) >= 6 {
		selectedInfo = m.selectedLocaleInfo.Render(fmt.Sprintf("selected locale=%s completion=%s", row[0], row[5]))
	}

	helpLine := m.help.View(m.keys)
	tableHelp := m.metaStyle.Render(m.tbl.HelpView())

	parts := []string{title, scope, overall, "", m.tbl.View()}
	if selectedInfo != "" {
		parts = append(parts, selectedInfo)
	}
	parts = append(parts, tableHelp, helpLine)

	return tea.NewView(strings.Join(parts, "\n"))
}

func (m *statusDashboardModel) sortRows() {
	sort.SliceStable(m.summary.ByLocale, func(i, j int) bool {
		li := m.summary.ByLocale[i]
		lj := m.summary.ByLocale[j]
		var less bool
		switch m.sortMode {
		case sortByUntranslated:
			if li.Untranslated == lj.Untranslated {
				less = li.Locale < lj.Locale
			} else {
				less = li.Untranslated > lj.Untranslated
			}
		case sortByTranslated:
			if li.Translated == lj.Translated {
				less = li.Locale < lj.Locale
			} else {
				less = li.Translated > lj.Translated
			}
		default:
			less = li.Locale < lj.Locale
		}
		if m.reverse {
			return !less
		}
		return less
	})

	rows := make([]table.Row, 0, len(m.summary.ByLocale))
	for _, row := range m.summary.ByLocale {
		rows = append(rows, table.Row{
			row.Locale,
			fmt.Sprintf("%d", row.Total),
			fmt.Sprintf("%d", row.Translated),
			fmt.Sprintf("%d", row.NeedsReview),
			fmt.Sprintf("%d", row.Untranslated),
			percent(row.Translated, row.Total),
		})
	}
	cursor := m.tbl.Cursor()
	m.tbl.SetRows(rows)
	if cursor >= len(rows) {
		cursor = len(rows) - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	m.tbl.SetCursor(cursor)
}

func (m statusDashboardModel) sortModeLabel() string {
	switch m.sortMode {
	case sortByUntranslated:
		return "untranslated"
	case sortByTranslated:
		return "translated"
	default:
		return "locale"
	}
}

func buildStatusSummary(entries []storage.Entry, locales []string) statusSummary {
	byLocale := make(map[string]*localeSummary, len(locales))
	for _, locale := range locales {
		byLocale[locale] = &localeSummary{Locale: locale}
	}

	s := statusSummary{}
	for _, entry := range entries {
		status := computeStatus(entry)
		s.Total++
		row, ok := byLocale[entry.Locale]
		if !ok {
			row = &localeSummary{Locale: entry.Locale}
			byLocale[entry.Locale] = row
		}
		row.Total++
		switch status {
		case "translated":
			s.Translated++
			row.Translated++
		case "needs_review":
			s.NeedsReview++
			row.NeedsReview++
		default:
			s.Untranslated++
			row.Untranslated++
		}
	}

	localesOut := make([]string, 0, len(byLocale))
	for locale := range byLocale {
		localesOut = append(localesOut, locale)
	}
	sort.Strings(localesOut)
	for _, locale := range localesOut {
		s.ByLocale = append(s.ByLocale, *byLocale[locale])
	}

	return s
}

func percent(numerator, denominator int) string {
	if denominator <= 0 {
		return "0.0%"
	}
	pct := (float64(numerator) / float64(denominator)) * 100
	return fmt.Sprintf("%.1f%%", pct)
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func isTTYWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
