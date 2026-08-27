package main

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var tuiBookmarks = []bookmark{
	{command: "lt", url: "https://logs.example.com/test?s={service}", description: "logs test alpha cloud"},
	{command: "lp", url: "https://logs.example.com/prod?s={service}", description: "logs prod alpha cloud"},
	{command: "board", url: "https://tracker.example.com/board", description: "issue board"},
	{url: "https://example.com/one", description: "widget history prod"},
	{url: "https://example.com/two", description: "widget history test"},
}

func send(t *testing.T, m model, msgs ...tea.Msg) (model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, msg := range msgs {
		var next tea.Model
		next, cmd = m.Update(msg)
		got, ok := next.(model)
		if !ok {
			t.Fatalf("Update returned %T, want model", next)
		}
		m = got
	}
	return m, cmd
}

func typeText(t *testing.T, m model, s string) model {
	t.Helper()
	for _, r := range s {
		m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestTUIFiltersAsYouType(t *testing.T) {
	m := initialModel(tuiBookmarks, "")
	if len(m.results) != len(tuiBookmarks) {
		t.Fatalf("empty query gave %d results, want all %d", len(m.results), len(tuiBookmarks))
	}

	m = typeText(t, m, "widget")
	if len(m.results) != 2 {
		t.Fatalf("got %d results for \"order\", want 2", len(m.results))
	}
	if view := m.View(); !strings.Contains(view, "widget history prod") {
		t.Errorf("view missing expected result:\n%s", view)
	}
}

func TestTUIPrefillsQuery(t *testing.T) {
	m := initialModel(tuiBookmarks, "board")
	if m.input.Value() != "board" {
		t.Errorf("input is %q, want \"board\"", m.input.Value())
	}
	if len(m.results) == 0 || m.results[0].bookmark.command != "board" {
		t.Errorf("expected !board ranked first, got %+v", descriptions(m.results))
	}
}

func TestTUICursorMovesAndClamps(t *testing.T) {
	m := initialModel(tuiBookmarks, "")
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("cursor went to %d above the top, want 0", m.cursor)
	}

	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("cursor is %d after one down, want 1", m.cursor)
	}

	for range len(tuiBookmarks) + 5 {
		m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if want := len(m.results) - 1; m.cursor != want {
		t.Errorf("cursor is %d past the bottom, want %d", m.cursor, want)
	}
}

func TestTUICtrlNAndCtrlPMove(t *testing.T) {
	m := initialModel(tuiBookmarks, "")
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if m.cursor != 1 {
		t.Errorf("ctrl+n left cursor at %d, want 1", m.cursor)
	}
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	if m.cursor != 0 {
		t.Errorf("ctrl+p left cursor at %d, want 0", m.cursor)
	}
}

func TestTUICursorResetsWhenQueryChanges(t *testing.T) {
	m := initialModel(tuiBookmarks, "")
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor == 0 {
		t.Fatal("cursor did not move")
	}

	m = typeText(t, m, "o")
	if m.cursor != 0 || m.offset != 0 {
		t.Errorf("cursor/offset are %d/%d after typing, want 0/0", m.cursor, m.offset)
	}
}

func TestTUIEnterOpensPlainBookmark(t *testing.T) {
	m := initialModel(tuiBookmarks, "issue board")
	m, cmd := send(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.chosen != "https://tracker.example.com/board" {
		t.Errorf("chosen is %q", m.chosen)
	}
	if !isQuit(cmd) {
		t.Error("expected the program to quit")
	}
}

func TestTUIEnterOnParameterBookmarkPrompts(t *testing.T) {
	m := initialModel(tuiBookmarks, "lp")
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.stage != stageParam {
		t.Fatal("expected the parameter stage")
	}
	if !strings.Contains(m.paramInput.Prompt, "service") {
		t.Errorf("prompt is %q, want it to name the parameter", m.paramInput.Prompt)
	}
	if m.chosen != "" {
		t.Errorf("chosen set too early: %q", m.chosen)
	}

	m = typeText(t, m, "nginx")
	m, cmd := send(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	want := "https://logs.example.com/prod?s=nginx"
	if m.chosen != want {
		t.Errorf("chosen is %q, want %q", m.chosen, want)
	}
	if !isQuit(cmd) {
		t.Error("expected the program to quit")
	}
}

func TestTUIEscFromParameterReturnsToSearch(t *testing.T) {
	m := initialModel(tuiBookmarks, "lp")
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.stage != stageSearch {
		t.Error("expected to be back at the search stage")
	}
	if m.input.Value() != "lp" {
		t.Errorf("query is %q, want it preserved as \"lp\"", m.input.Value())
	}
	if m.chosen != "" {
		t.Errorf("chosen should be empty, got %q", m.chosen)
	}
}

func TestTUIEscFromSearchQuitsWithoutOpening(t *testing.T) {
	m := initialModel(tuiBookmarks, "")
	m, cmd := send(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.chosen != "" {
		t.Errorf("chosen is %q, want empty", m.chosen)
	}
	if !isQuit(cmd) {
		t.Error("expected the program to quit")
	}
}

func TestTUIShowsURLForSelectedOnly(t *testing.T) {
	m := initialModel(tuiBookmarks, "widget")
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	if !strings.Contains(view, "example.com/one") {
		t.Errorf("selected result's URL missing:\n%s", view)
	}
	if strings.Contains(view, "example.com/two") {
		t.Errorf("unselected result's URL should be hidden:\n%s", view)
	}

	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	view = m.View()
	if !strings.Contains(view, "example.com/two") {
		t.Errorf("URL did not follow the cursor:\n%s", view)
	}
	if strings.Contains(view, "example.com/one") {
		t.Errorf("previous URL still shown:\n%s", view)
	}
}

func TestTUIScrollsToKeepCursorVisible(t *testing.T) {
	// Height 7 leaves 4 result rows.
	m := initialModel(tuiBookmarks, "")
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 7})
	if m.visibleRows() != 4 {
		t.Fatalf("visibleRows is %d, want 4", m.visibleRows())
	}

	for range len(tuiBookmarks) {
		m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.cursor < m.offset || m.cursor >= m.offset+m.visibleRows() {
		t.Errorf("cursor %d outside window [%d,%d)",
			m.cursor, m.offset, m.offset+m.visibleRows())
	}
	if !strings.Contains(m.View(), m.results[m.cursor].bookmark.description) {
		t.Error("selected result is not rendered")
	}
}

func TestTUINoMatches(t *testing.T) {
	m := initialModel(tuiBookmarks, "zzzznotfound")
	if len(m.results) != 0 {
		t.Fatalf("expected no results, got %d", len(m.results))
	}
	if !strings.Contains(m.View(), "No matches") {
		t.Errorf("view should say so:\n%s", m.View())
	}
	// Enter must be a no-op rather than a panic.
	m, cmd := send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.chosen != "" || isQuit(cmd) {
		t.Error("enter should do nothing with no results")
	}
}

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// The highlight bar is assembled from separately styled segments, each of
// which must repeat the background. One that forgets emits a reset and leaves
// a gap, which is invisible in a plain-text assertion.
func TestSelectedBarIsUnbroken(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(restore)

	m := initialModel(tuiBookmarks, "prod")
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})

	row := strings.Split(m.View(), "\n")[1]
	if got := lipgloss.Width(row); got != 40 {
		t.Errorf("bar is %d cols wide, want the full 40", got)
	}

	active := false
	pos := 0
	for _, loc := range ansiRe.FindAllStringIndex(row, -1) {
		if visible := row[pos:loc[0]]; visible != "" && !active {
			t.Errorf("gap in bar: %q has no background", visible)
		}
		switch code := row[loc[0]:loc[1]]; {
		case strings.Contains(code, "48;5;"):
			active = true
		case code == "\x1b[0m":
			active = false
		}
		pos = loc[1]
	}
	if tail := row[pos:]; tail != "" && !active {
		t.Errorf("gap at end of bar: %q", tail)
	}
}

func TestUnselectedRowsHaveNoBar(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(restore)

	m := initialModel(tuiBookmarks, "prod")
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 40, Height: 12})

	// Line 0 is the prompt, 1 the selected row, 2 its URL, 3 the next result.
	if row := strings.Split(m.View(), "\n")[3]; strings.Contains(row, "48;5;") {
		t.Errorf("unselected row carries a background: %q", row)
	}
}

func TestMatchedCharactersAreColoured(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(restore)

	plain := highlightMatches("logs prod", nil)
	if ansiRe.MatchString(plain) {
		t.Errorf("unmatched text should carry no styling, got %q", plain)
	}
	hit := highlightMatches("logs prod", []int{0, 1, 2, 3})
	if !ansiRe.MatchString(hit) {
		t.Errorf("matched text should be styled, got %q", hit)
	}
	if ansiRe.ReplaceAllString(hit, "") != "logs prod" {
		t.Errorf("styling changed the text: %q", hit)
	}
}
