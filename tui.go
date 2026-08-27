package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type stage int

const (
	stageSearch stage = iota
	stageParam
)

var helpStyle = lipgloss.NewStyle().Faint(true)

// chromeLines is what the view spends on things other than result rows: the
// search prompt, the URL under the cursor, and the footer.
const chromeLines = 3

type model struct {
	bookmarks []bookmark
	input     textinput.Model
	results   []searchResult
	cursor    int // index into results
	offset    int // first visible result
	width     int
	height    int

	stage      stage
	paramInput textinput.Model
	pending    string // url awaiting a {param} value
	chosen     string // url to open; set just before quitting
}

func initialModel(bookmarks []bookmark, query string) model {
	in := textinput.New()
	in.Prompt = "Search: "
	in.SetValue(query)
	in.CursorEnd()
	in.Focus()

	param := textinput.New()

	m := model{
		bookmarks: bookmarks,
		input:     in,
		// Sane defaults until the first WindowSizeMsg arrives.
		width:      80,
		height:     24,
		paramInput: param,
	}
	m.results = search(bookmarks, query)
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// visibleRows is how many result rows fit in the window.
func (m model) visibleRows() int {
	n := m.height - chromeLines
	if n < 1 {
		return 1
	}
	return n
}

// clampCursor keeps the cursor inside the results and the scroll window around
// the cursor.
func (m *model) clampCursor() {
	if len(m.results) == 0 {
		m.cursor, m.offset = 0, 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
	}
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// selectCurrent moves toward opening the highlighted result: straight to quit
// when the URL is complete, or into the parameter stage when it is not.
func (m model) selectCurrent() (tea.Model, tea.Cmd) {
	if len(m.results) == 0 {
		return m, nil
	}
	url := m.results[m.cursor].bookmark.url
	if !hasParameter(url) {
		m.chosen = url
		return m, tea.Quit
	}
	m.stage = stageParam
	m.pending = url
	m.paramInput.Prompt = fmt.Sprintf("Enter %s: ", parameterName(url))
	m.paramInput.SetValue("")
	m.paramInput.Focus()
	m.input.Blur()
	return m, textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampCursor()
		return m, nil

	case tea.KeyMsg:
		if m.stage == stageParam {
			return m.updateParam(msg)
		}
		return m.updateSearch(msg)
	}
	return m, nil
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "ctrl+p":
		m.cursor--
		m.clampCursor()
		return m, nil
	case "down", "ctrl+n":
		m.cursor++
		m.clampCursor()
		return m, nil
	case "enter":
		return m.selectCurrent()
	}

	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.results = search(m.bookmarks, m.input.Value())
		m.cursor, m.offset = 0, 0
	}
	return m, cmd
}

func (m model) updateParam(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		// Back to the list, query untouched.
		m.stage = stageSearch
		m.pending = ""
		m.paramInput.Blur()
		m.input.Focus()
		return m, textinput.Blink
	case "enter":
		m.chosen = resolveURL(m.pending, strings.TrimSpace(m.paramInput.Value()))
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.paramInput, cmd = m.paramInput.Update(msg)
	return m, cmd
}

// padRow extends the highlight bar to the full terminal width. lipgloss.Width
// measures visible columns, ignoring the ANSI codes already in the row.
func padRow(row string, width int) string {
	if gap := width - lipgloss.Width(row); gap > 0 {
		return row + selectedStyle.Render(strings.Repeat(" ", gap))
	}
	return row
}

func (m model) View() string {
	if m.stage == stageParam {
		return m.input.View() + "\n" + m.paramInput.View() + "\n"
	}

	var sb strings.Builder
	sb.WriteString(m.input.View())
	sb.WriteString("\n")

	if len(m.results) == 0 {
		sb.WriteString(helpStyle.Render("  No matches"))
		sb.WriteString("\n")
		return sb.String()
	}

	end := m.offset + m.visibleRows()
	if end > len(m.results) {
		end = len(m.results)
	}
	for i := m.offset; i < end; i++ {
		r := m.results[i]
		n := fmt.Sprintf("%2d", i+1)
		desc := r.bookmark.displayText()

		if i == m.cursor {
			row := selectedStyle.Render("▸ ") + selectedNum.Render(n) + selectedStyle.Render(" ") +
				highlightMatchesWith(desc, r.matchedIndexes, selectedStyle, selectedMatch)
			sb.WriteString(padRow(row, m.width) + "\n")
			sb.WriteString(formatURL(r.bookmark.url, m.width) + "\n")
		} else {
			sb.WriteString("  " + numberStyle.Render(n) + " " +
				highlightMatchesWith(desc, r.matchedIndexes, lipgloss.NewStyle(), matchStyle) + "\n")
		}
	}

	sb.WriteString(helpStyle.Render(fmt.Sprintf(
		"  %d/%d   ↑↓ move · enter open · esc quit", m.cursor+1, len(m.results))))
	sb.WriteString("\n")
	return sb.String()
}
