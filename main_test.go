package main

import (
	"strings"
	"testing"
)

func TestParseBookmarks(t *testing.T) {
	input := `https://example.com/help - Description about example
https://example.org/ - This is a webpage about examples
# dev tools
https://example.net

`
	bookmarks := parseBookmarks(strings.NewReader(input))

	if len(bookmarks) != 3 {
		t.Fatalf("expected 3 bookmarks, got %d", len(bookmarks))
	}
	if bookmarks[0].url != "https://example.com/help" {
		t.Errorf("expected url 'https://example.com/help', got '%s'", bookmarks[0].url)
	}
	if bookmarks[0].description != "Description about example" {
		t.Errorf("expected description 'Description about example', got '%s'", bookmarks[0].description)
	}
	if bookmarks[2].url != "https://example.net" {
		t.Errorf("expected url 'https://example.net', got '%s'", bookmarks[2].url)
	}
	if bookmarks[2].description != "" {
		t.Errorf("expected empty description, got '%s'", bookmarks[2].description)
	}
}

func TestParseBookmarksDescriptionWithDash(t *testing.T) {
	input := `https://example.com - foo - bar - baz`
	bookmarks := parseBookmarks(strings.NewReader(input))

	if len(bookmarks) != 1 {
		t.Fatalf("expected 1 bookmark, got %d", len(bookmarks))
	}
	if bookmarks[0].url != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got '%s'", bookmarks[0].url)
	}
	if bookmarks[0].description != "foo - bar - baz" {
		t.Errorf("expected description 'foo - bar - baz', got '%s'", bookmarks[0].description)
	}
}

func TestParseBookmarksEmpty(t *testing.T) {
	input := `# only comments
# and blank lines

`
	bookmarks := parseBookmarks(strings.NewReader(input))
	if len(bookmarks) != 0 {
		t.Fatalf("expected 0 bookmarks, got %d", len(bookmarks))
	}
}

func TestSearch(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com/help", description: "Description about one thing"},
		{url: "https://example.org/", description: "This is a webpage about bones"},
		{url: "https://example.net", description: ""},
	}

	results := search(bookmarks, "one")
	if len(results) < 1 {
		t.Fatal("expected at least 1 result")
	}
	// Both "one thing" and "bones" should match "one"
	if len(results) < 2 {
		t.Fatal("expected at least 2 results")
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com", description: "Code Hosting Service"},
	}

	results := search(bookmarks, "code hosting")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestSearchNoResults(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com/help", description: "Description about example"},
	}

	results := search(bookmarks, "zzzznotfound")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestParameterName(t *testing.T) {
	name := parameterName("http://example.com/search/{query}")
	if name != "query" {
		t.Errorf("expected 'query', got '%s'", name)
	}
	name = parameterName("http://example.com/page")
	if name != "" {
		t.Errorf("expected empty, got '%s'", name)
	}
}

func TestHasParameter(t *testing.T) {
	if !hasParameter("http://example.com/{query}") {
		t.Error("expected true for URL with parameter")
	}
	if hasParameter("http://example.com/page") {
		t.Error("expected false for URL without parameter")
	}
}

func TestResolveURL(t *testing.T) {
	url := resolveURL("http://example.com/search/{query}", "hello")
	if url != "http://example.com/search/hello" {
		t.Errorf("expected 'http://example.com/search/hello', got '%s'", url)
	}
}

func TestResolveURLNoParameter(t *testing.T) {
	url := resolveURL("http://example.com/page", "ignored")
	if url != "http://example.com/page" {
		t.Errorf("expected 'http://example.com/page', got '%s'", url)
	}
}

func TestParseSelection(t *testing.T) {
	n, arg := parseSelection("2 SE0000108656")
	if n != 2 {
		t.Errorf("expected selection 2, got %d", n)
	}
	if arg != "SE0000108656" {
		t.Errorf("expected arg 'SE0000108656', got '%s'", arg)
	}
}

func TestParseSelectionNoArg(t *testing.T) {
	n, arg := parseSelection("3")
	if n != 3 {
		t.Errorf("expected selection 3, got %d", n)
	}
	if arg != "" {
		t.Errorf("expected empty arg, got '%s'", arg)
	}
}

func TestParseSelectionTrimArg(t *testing.T) {
	n, arg := parseSelection("2  SE0000108656  ")
	if n != 2 {
		t.Errorf("expected selection 2, got %d", n)
	}
	if arg != "SE0000108656" {
		t.Errorf("expected arg 'SE0000108656', got '%s'", arg)
	}
}

func TestParseSelectionInvalid(t *testing.T) {
	n, _ := parseSelection("abc")
	if n != 0 {
		t.Errorf("expected 0 for invalid input, got %d", n)
	}
}

func TestSearchMatchesOnDescriptionOnly(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com/boards/1629", description: "board jira"},
		{url: "https://example.com/logs/query;query=app%3D%22service-{query}", description: "lpo logs test onprem"},
	}
	results := search(bookmarks, "lpo")
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].bookmark.description != "lpo logs test onprem" {
		t.Errorf("expected 'lpo logs test onprem' as top result, got '%s'", results[0].bookmark.description)
	}
}

func TestSearchDoesNotMatchURL(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com/boards/1629", description: "board jira"},
	}
	results := search(bookmarks, "boards")
	if len(results) != 0 {
		t.Errorf("expected 0 results when query matches URL but not description, got %d", len(results))
	}
}

func TestSearchFallsBackToURLWhenNoDescription(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com"},
	}
	results := search(bookmarks, "example")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestParseBookmarksWithCommand(t *testing.T) {
	input := `!pr https://example.com/pulls/{id} - Pull requests
!dash https://example.com/dashboard - Dashboard
https://example.com/docs - Documentation`
	bookmarks := parseBookmarks(strings.NewReader(input))

	if len(bookmarks) != 3 {
		t.Fatalf("expected 3 bookmarks, got %d", len(bookmarks))
	}
	if bookmarks[0].command != "pr" {
		t.Errorf("expected command 'pr', got '%s'", bookmarks[0].command)
	}
	if bookmarks[0].url != "https://example.com/pulls/{id}" {
		t.Errorf("unexpected url: %s", bookmarks[0].url)
	}
	if bookmarks[0].description != "Pull requests" {
		t.Errorf("unexpected description: %s", bookmarks[0].description)
	}
	if bookmarks[1].command != "dash" {
		t.Errorf("expected command 'dash', got '%s'", bookmarks[1].command)
	}
	if bookmarks[2].command != "" {
		t.Errorf("expected no command, got '%s'", bookmarks[2].command)
	}
}

func TestFindCommand(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com/{query}", description: "Example", command: "ex"},
		{url: "https://example.com/other", description: "Other"},
	}
	cmd := findCommand(bookmarks, "ex")
	if cmd == nil {
		t.Fatal("expected to find command 'ex'")
	}
	if cmd.url != "https://example.com/{query}" {
		t.Errorf("unexpected url: %s", cmd.url)
	}
}

func TestFindCommandCaseInsensitive(t *testing.T) {
	bookmarks := []bookmark{
		{url: "http://example.com", description: "Example", command: "EX"},
	}
	cmd := findCommand(bookmarks, "ex")
	if cmd == nil {
		t.Fatal("expected to find command 'EX' with query 'ex'")
	}
}

func TestFindCommandNotFound(t *testing.T) {
	bookmarks := []bookmark{
		{url: "http://example.com", description: "Example", command: "ex"},
	}
	cmd := findCommand(bookmarks, "notfound")
	if cmd != nil {
		t.Error("expected nil for unknown command")
	}
}

func TestCommandBookmarksAppearInSearch(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com/admin/{id}", description: "Admin panel", command: "adm"},
		{url: "https://example.com/admin/docs", description: "Admin docs"},
	}
	results := search(bookmarks, "admin")
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
}

func TestHighlightMatches(t *testing.T) {
	result := highlightMatches("hello", []int{0, 1})
	if !strings.Contains(result, "he") {
		t.Error("expected matched chars in output")
	}
	// Note: lipgloss may not render ANSI codes in test environment,
	// so we just verify the function runs and contains expected text
}

func TestHighlightMatchesEmpty(t *testing.T) {
	result := highlightMatches("hello", nil)
	if result != "hello" {
		t.Errorf("expected plain 'hello' with no matches, got '%s'", result)
	}
}

func TestFormatResults(t *testing.T) {
	results := []searchResult{
		{bookmark: bookmark{url: "https://example.com/help", description: "Description about example"}, matchedIndexes: nil},
		{bookmark: bookmark{url: "https://example.net", description: ""}, matchedIndexes: nil},
	}

	output := formatResults(results, 80)
	if !strings.Contains(output, "Description about example") {
		t.Error("expected description in output")
	}
	if !strings.Contains(output, "https://example.com/help") {
		t.Error("expected URL shown below description")
	}
	if !strings.Contains(output, "https://example.net") {
		t.Error("expected URL in output when no description")
	}
}

func TestTruncateMiddle(t *testing.T) {
	short := "https://example.com"
	if truncateMiddle(short, 80) != short {
		t.Error("short URL should not be truncated")
	}

	long := "https://example.com/very/long/path/that/goes/on/and/on/and/on/forever"
	result := truncateMiddle(long, 30)
	if len(result) > 30 {
		t.Errorf("expected length <= 30, got %d", len(result))
	}
	if !strings.Contains(result, "...") {
		t.Error("expected ellipsis in truncated URL")
	}
}

func TestHostEndIndex(t *testing.T) {
	if idx := hostEndIndex("https://example.com/path"); idx != 19 {
		t.Errorf("expected 19, got %d", idx)
	}
	if idx := hostEndIndex("https://example.com"); idx != -1 {
		t.Errorf("expected -1 for URL without path, got %d", idx)
	}
	if idx := hostEndIndex("not-a-url"); idx != -1 {
		t.Errorf("expected -1 for non-URL, got %d", idx)
	}
}

func TestFormatURLNoParam(t *testing.T) {
	short := "https://example.com/page"
	result := formatURL(short, 80)
	if !strings.Contains(result, short) {
		t.Error("short URL should be shown in full")
	}
}

func TestFormatURLWithParam(t *testing.T) {
	url := "https://example.com/logs/query;query=resource.labels.container_name%20%3D%20%22{service}%22;storageScope=storage,projects/example-project"
	result := formatURL(url, 80)
	if !strings.Contains(result, "{service}") {
		t.Error("expected {service} in output")
	}
	if !strings.Contains(result, "...") {
		t.Error("expected truncation ellipsis")
	}
}

func TestFormatURLShortWithParam(t *testing.T) {
	url := "https://example.com/{query}"
	result := formatURL(url, 80)
	if !strings.Contains(result, "{query}") {
		t.Error("expected {query} in output")
	}
	if !strings.Contains(result, "https://example.com/") {
		t.Error("expected full URL when it fits")
	}
}
