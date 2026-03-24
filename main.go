package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/sahilm/fuzzy"
)

type bookmark struct {
	url         string
	description string
	command     string // e.g. "tpo" from a "!tpo" prefix
}

var (
	numberStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	dimStyle    = lipgloss.NewStyle().Faint(true)
	boldStyle   = lipgloss.NewStyle().Bold(true)
)

func highlightMatches(text string, matchedIndexes []int) string {
	if len(matchedIndexes) == 0 {
		return text
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
			sb.WriteString(boldStyle.Render(string(run)))
		} else {
			var run []rune
			for i < len(runes) && !matched[i] {
				run = append(run, runes[i])
				i++
			}
			sb.WriteString(string(run))
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

func (b bookmarkSource) String(i int) string {
	if b[i].description != "" {
		return strings.ToLower(b[i].description)
	}
	return strings.ToLower(b[i].url)
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

func search(bookmarks []bookmark, query string) []searchResult {
	matches := fuzzy.FindFrom(strings.ToLower(query), bookmarkSource(bookmarks))
	var results []searchResult
	for _, m := range matches {
		results = append(results, searchResult{
			bookmark:       bookmarks[m.Index],
			matchedIndexes: m.MatchedIndexes,
		})
	}
	return results
}

func terminalWidth() int {
	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		return 80
	}
	return w
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

func formatResults(results []searchResult, width int) string {
	var sb strings.Builder
	for i, r := range results {
		num := numberStyle.Render(fmt.Sprintf("%d", i+1))
		sep := dimStyle.Render(")")

		var text string
		if r.bookmark.description != "" {
			text = highlightMatches(r.bookmark.description, r.matchedIndexes)
		} else {
			text = highlightMatches(r.bookmark.url, r.matchedIndexes)
		}

		fmt.Fprintf(&sb, "%s%s %s\n", num, sep, text)
		sb.WriteString(formatURL(r.bookmark.url, width))
		sb.WriteString("\n\n")
	}
	return sb.String()
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

func parseSelection(input string) (int, string) {
	parts := strings.SplitN(input, " ", 2)
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, ""
	}
	arg := ""
	if len(parts) == 2 {
		arg = strings.TrimSpace(parts[1])
	}
	return n, arg
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
	// Clean exit on ctrl-c
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		fmt.Println()
		os.Exit(0)
	}()

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

	scanner := bufio.NewScanner(os.Stdin)
	query := strings.Join(os.Args[1:], " ")

	// Check for command match on first arg
	if len(os.Args) >= 2 {
		if cmd := findCommand(bookmarks, os.Args[1]); cmd != nil {
			url := cmd.url
			arg := strings.TrimSpace(strings.Join(os.Args[2:], " "))
			if hasParameter(url) {
				if arg == "" {
					fmt.Printf("Enter %s: ", parameterName(url))
					if !scanner.Scan() {
						return
					}
					arg = strings.TrimSpace(scanner.Text())
				}
				url = resolveURL(url, arg)
			}
			if err := exec.Command("open", url).Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open URL: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	for {
		// Get query if not provided
		for query == "" {
			fmt.Print("Search: ")
			if !scanner.Scan() {
				return
			}
			query = strings.TrimSpace(scanner.Text())
		}

		results := search(bookmarks, query)
		if len(results) == 0 {
			fmt.Println("No matches")
			query = ""
			continue
		}

		fmt.Print(formatResults(results, terminalWidth()))

		// Selection loop
		for {
			fmt.Print("\nWhere to go? ")
			if !scanner.Scan() {
				return
			}
			input := strings.TrimSpace(scanner.Text())
			n, arg := parseSelection(input)
			if n < 1 || n > len(results) {
				continue
			}
			url := results[n-1].bookmark.url
			if hasParameter(url) {
				if arg == "" {
					fmt.Printf("Enter %s: ", parameterName(url))
					if !scanner.Scan() {
						return
					}
					arg = strings.TrimSpace(scanner.Text())
				}
				url = resolveURL(url, arg)
			}
			if err := exec.Command("open", url).Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open URL: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}
}
