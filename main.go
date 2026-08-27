package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/sahilm/fuzzy"
)

type bookmark struct {
	url         string
	description string
	command     string // e.g. "tpo" from a "!tpo" prefix
}

// selectedBg is the highlight bar behind the cursor row. Every style used on
// that row repeats it, so the bar stays unbroken across styled segments.
var selectedBg = lipgloss.AdaptiveColor{Light: "153", Dark: "24"}

var (
	numberStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	dimStyle    = lipgloss.NewStyle().Faint(true)
	boldStyle   = lipgloss.NewStyle().Bold(true)

	// matchStyle picks out the fuzzy-matched characters.
	matchStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "5", Dark: "13"})

	selectedStyle = lipgloss.NewStyle().Background(selectedBg)
	selectedNum   = numberStyle.Background(selectedBg)
	selectedMatch = matchStyle.Background(selectedBg)
)

func highlightMatches(text string, matchedIndexes []int) string {
	return highlightMatchesWith(text, matchedIndexes, lipgloss.NewStyle(), matchStyle)
}

// highlightMatchesWith renders text with the matched characters picked out.
// Both styles are passed in so a selected row can carry its background colour
// through every segment — a style that omitted it would emit a reset and punch
// a hole in the highlight bar.
func highlightMatchesWith(text string, matchedIndexes []int, base, hit lipgloss.Style) string {
	if len(matchedIndexes) == 0 {
		return base.Render(text)
	}
	matched := make(map[int]bool, len(matchedIndexes))
	for _, idx := range matchedIndexes {
		matched[idx] = true
	}

	var sb strings.Builder
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		if matched[i] {
			var run []rune
			for i < len(runes) && matched[i] {
				run = append(run, runes[i])
				i++
			}
			sb.WriteString(hit.Render(string(run)))
		} else {
			var run []rune
			for i < len(runes) && !matched[i] {
				run = append(run, runes[i])
				i++
			}
			sb.WriteString(base.Render(string(run)))
		}
	}
	return sb.String()
}

func parseBookmarks(r io.Reader) []bookmark {
	var bookmarks []bookmark
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		b := bookmark{}
		// Parse optional !command prefix
		if strings.HasPrefix(line, "!") {
			spaceIdx := strings.Index(line, " ")
			if spaceIdx == -1 {
				continue // "!cmd" with no URL, skip
			}
			b.command = line[1:spaceIdx]
			line = line[spaceIdx+1:]
		}
		if idx := strings.Index(line, " - "); idx != -1 {
			b.url = line[:idx]
			b.description = line[idx+3:]
		} else {
			b.url = line
		}
		bookmarks = append(bookmarks, b)
	}
	return bookmarks
}

type bookmarkSource []bookmark

// displayText is what the UI shows for a bookmark, and the coordinate space
// its matchedIndexes refer to.
func (b bookmark) displayText() string {
	if b.description != "" {
		return b.description
	}
	return b.url
}

// String prefixes the command keyword so that "dvl" ranks its own bookmark,
// and keeps helping in longer queries like "dvl prod".
func (b bookmarkSource) String(i int) string {
	text := b[i].displayText()
	if b[i].command != "" {
		text = b[i].command + " " + text
	}
	return strings.ToLower(text)
}

func (b bookmarkSource) Len() int {
	return len(b)
}

func findCommand(bookmarks []bookmark, keyword string) *bookmark {
	keyword = strings.ToLower(keyword)
	for i := range bookmarks {
		if bookmarks[i].command != "" && strings.ToLower(bookmarks[i].command) == keyword {
			return &bookmarks[i]
		}
	}
	return nil
}

type searchResult struct {
	bookmark       bookmark
	matchedIndexes []int
}

// search matches each whitespace-separated term independently, so term order
// does not affect the outcome: "logs prod alpha" and "logs alpha prod" return the
// same bookmarks in the same order. A bookmark must match every term.
func search(bookmarks []bookmark, query string) []searchResult {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		// No query is no filter: show everything, in file order.
		results := make([]searchResult, 0, len(bookmarks))
		for _, b := range bookmarks {
			results = append(results, searchResult{bookmark: b})
		}
		return results
	}

	src := bookmarkSource(bookmarks)
	type accumulator struct {
		bonus   int // score with fuzzy's per-call length penalty backed out
		matched int // number of characters matched across all terms
		indexes []int
	}
	current := make(map[int]*accumulator)
	for i, term := range terms {
		next := make(map[int]*accumulator)
		for _, m := range fuzzy.FindFrom(term, src) {
			// fuzzy charges len(MatchedIndexes)-len(str) once per call.
			// Summing raw scores would charge it once per term, letting
			// candidate length outweigh match quality. Back it out here and
			// apply it a single time below.
			bonus := m.Score - (len(m.MatchedIndexes) - len(src.String(m.Index)))
			if i == 0 {
				next[m.Index] = &accumulator{
					bonus:   bonus,
					matched: len(m.MatchedIndexes),
					indexes: m.MatchedIndexes,
				}
				continue
			}
			if prev, ok := current[m.Index]; ok {
				next[m.Index] = &accumulator{
					bonus:   prev.bonus + bonus,
					matched: prev.matched + len(m.MatchedIndexes),
					indexes: append(prev.indexes, m.MatchedIndexes...),
				}
			}
		}
		current = next
		if len(current) == 0 {
			return nil
		}
	}

	scores := make(map[int]int, len(current))
	indexes := make([]int, 0, len(current))
	for idx, a := range current {
		scores[idx] = a.bonus - (len(src.String(idx)) - a.matched)
		indexes = append(indexes, idx)
	}
	// Best score first, falling back to file order so output stays stable.
	sort.Slice(indexes, func(i, j int) bool {
		a, b := indexes[i], indexes[j]
		if scores[a] != scores[b] {
			return scores[a] > scores[b]
		}
		return a < b
	})

	results := make([]searchResult, 0, len(indexes))
	for _, idx := range indexes {
		results = append(results, searchResult{
			bookmark:       bookmarks[idx],
			matchedIndexes: displayIndexes(bookmarks[idx], terms),
		})
	}
	return results
}

// displayIndexes recomputes which characters to highlight against the text
// actually rendered. Scoring matches the keyword-prefixed string so that "dvl"
// ranks its own bookmark, but fuzzy is then free to satisfy a term using the
// hidden keyword — "log" can match l and o inside "dvl" and only g in "logs".
// Translating those positions would light up a misleading fragment, so the
// terms are re-matched against the description on its own.
func displayIndexes(b bookmark, terms []string) []int {
	text := strings.ToLower(b.displayText())
	var out []int
	for _, term := range terms {
		if ms := fuzzy.Find(term, []string{text}); len(ms) > 0 {
			out = append(out, ms[0].MatchedIndexes...)
		}
	}
	return out
}
func hostEndIndex(url string) int {
	schemeEnd := strings.Index(url, "://")
	if schemeEnd == -1 {
		return -1
	}
	pathStart := strings.Index(url[schemeEnd+3:], "/")
	if pathStart == -1 {
		return -1
	}
	return schemeEnd + 3 + pathStart
}

func truncateMiddle(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	half := (maxWidth - 3) / 2
	if half < 1 {
		half = 1
	}
	return s[:half] + "..." + s[len(s)-half:]
}

func formatURL(url string, maxWidth int) string {
	const indent = "   "
	available := maxWidth - len(indent)
	if available < 10 {
		available = 10
	}

	loc := paramRegexp.FindStringIndex(url)

	// No parameter — show full or truncate middle
	if loc == nil {
		if len(url) <= available {
			return indent + dimStyle.Render(url)
		}
		return indent + dimStyle.Render(truncateMiddle(url, available))
	}

	// URL with parameter fits — show full with bold param
	if len(url) <= available {
		before := url[:loc[0]]
		paramText := url[loc[0]:loc[1]]
		after := url[loc[1]:]
		return indent + dimStyle.Render(before) + boldStyle.Render(paramText) + dimStyle.Render(after)
	}

	// Has parameter, needs truncation — show context around {param}
	paramText := url[loc[0]:loc[1]]

	beforeStart := loc[0] - 10
	if beforeStart < 0 {
		beforeStart = 0
	}
	afterEnd := loc[1] + 5
	if afterEnd > len(url) {
		afterEnd = len(url)
	}

	before := url[beforeStart:loc[0]]
	after := url[loc[1]:afterEnd]

	prefix := "..."
	if beforeStart == 0 {
		prefix = ""
	}
	suffix := "..."
	if afterEnd == len(url) {
		suffix = ""
	}

	// Try to include hostname
	hostEnd := hostEndIndex(url)
	if hostEnd > 0 && hostEnd < beforeStart {
		withHost := url[:hostEnd] + "..." + before + paramText + after + suffix
		if len(withHost) <= available {
			return indent + dimStyle.Render(url[:hostEnd]+"..."+before) + boldStyle.Render(paramText) + dimStyle.Render(after+suffix)
		}
	}

	return indent + dimStyle.Render(prefix+before) + boldStyle.Render(paramText) + dimStyle.Render(after+suffix)
}

var paramRegexp = regexp.MustCompile(`\{([^}]+)\}`)

func parameterName(url string) string {
	m := paramRegexp.FindStringSubmatch(url)
	if m != nil {
		return m[1]
	}
	return ""
}

func hasParameter(url string) bool {
	return paramRegexp.MatchString(url)
}

func resolveURL(url, arg string) string {
	return paramRegexp.ReplaceAllLiteralString(url, arg)
}

func loadBookmarksFile() ([]bookmark, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(home, ".bookmarks"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseBookmarks(f), nil
}

func main() {
	bookmarks, err := loadBookmarksFile()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not open ~/.bookmarks")
		fmt.Fprintln(os.Stderr, "Create a bookmarks file at ~/.bookmarks with one URL per line:")
		fmt.Fprintln(os.Stderr, "  https://example.com - My bookmark")
		os.Exit(1)
	}
	if len(bookmarks) == 0 {
		fmt.Fprintln(os.Stderr, "No bookmarks found in ~/.bookmarks")
		os.Exit(1)
	}

	if !term.IsTerminal(os.Stdin.Fd()) {
		fmt.Fprintln(os.Stderr, "bm requires an interactive terminal")
		os.Exit(1)
	}

	final, err := tea.NewProgram(initialModel(bookmarks, initialQuery(bookmarks, os.Args[1:]))).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Open only once the terminal has been restored.
	m, ok := final.(model)
	if !ok || m.chosen == "" {
		return
	}
	if err := exec.Command("open", m.chosen).Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open URL: %v\n", err)
		os.Exit(1)
	}
}

// initialQuery prefills the search box from the command line. A leading
// command keyword stands alone: "bm dvl web" searches for "dvl", and the
// service name is retyped at the parameter prompt.
func initialQuery(bookmarks []bookmark, args []string) string {
	if len(args) == 0 {
		return ""
	}
	if findCommand(bookmarks, args[0]) != nil {
		return args[0]
	}
	return strings.Join(args, " ")
}
