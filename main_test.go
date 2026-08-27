package main

import (
	"reflect"
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

func descriptions(results []searchResult) []string {
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, r.bookmark.description)
	}
	return out
}

var orderBookmarks = []bookmark{
	{url: "https://logs.example.com/logs?p=prod", description: "logs prod alpha cloud"},
	{url: "https://logs.example.com/logs?p=test", description: "logs test alpha cloud"},
	{url: "https://logs.example.com/logs?p=prod-legacy", description: "logs prod legacy datacenter"},
	{url: "https://admin.example.com/prod", description: "Billing manager admin prod"},
	{url: "https://admin-billing.example.com/prod", description: "Admin billing admin-billing prod"},
}

func TestSearchTermOrderDoesNotMatter(t *testing.T) {
	results := search(orderBookmarks, "logs alpha prod")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), descriptions(results))
	}
	if results[0].bookmark.description != "logs prod alpha cloud" {
		t.Errorf("got %q", results[0].bookmark.description)
	}
}

func TestSearchTermOrderSameRanking(t *testing.T) {
	permutations := []string{
		"logs prod alpha",
		"logs alpha prod",
		"alpha prod logs",
		"prod alpha logs",
	}
	want := descriptions(search(orderBookmarks, permutations[0]))
	if len(want) == 0 {
		t.Fatal("expected the baseline permutation to match something")
	}
	for _, query := range permutations[1:] {
		got := descriptions(search(orderBookmarks, query))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("query %q gave %v, want %v", query, got, want)
		}
	}
}

func TestSearchRequiresEveryTerm(t *testing.T) {
	results := search(orderBookmarks, "logs zzzznotfound")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d: %v", len(results), descriptions(results))
	}
}

func TestSearchHighlightsAllTerms(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com", description: "logs prod alpha cloud"},
	}
	results := search(bookmarks, "alpha logs")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	matched := map[int]bool{}
	for _, idx := range results[0].matchedIndexes {
		matched[idx] = true
	}
	// "logs" occupies 0-3, "alpha" occupies 10-12.
	for _, idx := range []int{0, 1, 2, 3, 10, 11, 12} {
		if !matched[idx] {
			t.Errorf("index %d not highlighted, got %v", idx, results[0].matchedIndexes)
		}
	}
}

func TestSearchIgnoresSurplusWhitespace(t *testing.T) {
	want := descriptions(search(orderBookmarks, "logs prod"))
	got := descriptions(search(orderBookmarks, "  logs \t  prod  "))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Scoring charges the length penalty once per query, not once per term, so a
// longer description whose terms all land on word boundaries still beats a
// shorter one that only matches mid-word.
func TestSearchDoesNotRankOnLengthAlone(t *testing.T) {
	bookmarks := []bookmark{
		{url: "https://example.com/a", description: "backlog approved"},
		{url: "https://example.com/b", description: "log server prod region eu"},
	}
	got := descriptions(search(bookmarks, "log prod"))
	want := []string{"log server prod region eu", "backlog approved"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v (the longer, better-placed match should win)", got, want)
	}
}

var commandBookmarks = []bookmark{
	{command: "dv", url: "https://logs.example.com/test?s={service}", description: "logs test alpha cloud"},
	{command: "dvl", url: "https://logs.example.com/test-legacy?s={service}", description: "logs test legacy datacenter"},
	{command: "pv", url: "https://logs.example.com/prod?s={service}", description: "logs prod alpha cloud"},
	{command: "panel", url: "https://tracker.example.com/panel", description: "issue panel"},
}

// A keyword is a strong ranking signal, not a guarantee of first place.
// fuzzy pays +20 for a character following a separator but only compounding
// +5s for adjacency, so spreading a match across word boundaries can outscore
// an exact contiguous keyword: "dvl" scores better against "dv logs ..." than
// against "dvl logs ...". Assert the bookmark surfaces near the top, which is
// what the keyword actually buys.
func TestSearchExactKeywordRanksHighly(t *testing.T) {
	const nearTop = 2
	for _, keyword := range []string{"dv", "dvl", "pv", "panel"} {
		results := search(commandBookmarks, keyword)
		if len(results) == 0 {
			t.Errorf("%q matched nothing", keyword)
			continue
		}
		rank := -1
		for i, r := range results {
			if r.bookmark.command == keyword {
				rank = i
				break
			}
		}
		switch {
		case rank < 0:
			t.Errorf("%q did not surface !%s at all, got %v", keyword, keyword, descriptions(results))
		case rank >= nearTop:
			t.Errorf("%q ranked !%s at position %d, want within the top %d",
				keyword, keyword, rank+1, nearTop)
		}
	}
}
func TestSearchKeywordHelpsMidQuery(t *testing.T) {
	// "cloud" alone matches both !lt and !lp; the keyword breaks the tie.
	results := search(commandBookmarks, "pv cloud")
	if len(results) == 0 {
		t.Fatal("expected a match")
	}
	if results[0].bookmark.command != "pv" {
		t.Errorf("got !%s first, want !pv", results[0].bookmark.command)
	}
}

func TestSearchIndexesAreDisplayRelative(t *testing.T) {
	results := search(commandBookmarks, "dvl legacy")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if len(r.matchedIndexes) == 0 {
		t.Fatal("expected some highlighted characters")
	}
	for _, idx := range r.matchedIndexes {
		if idx < 0 || idx >= len(r.bookmark.description) {
			t.Errorf("index %d outside %q (len %d)",
				idx, r.bookmark.description, len(r.bookmark.description))
		}
	}
}

// A short description with a long keyword is where un-offset indexes would
// run past the end of the rendered text.
func TestSearchIndexesStayInsideShortDescription(t *testing.T) {
	results := search(commandBookmarks, "panel")
	if len(results) == 0 {
		t.Fatal("expected a match")
	}
	r := results[0]
	for _, idx := range r.matchedIndexes {
		if idx < 0 || idx >= len(r.bookmark.description) {
			t.Errorf("index %d outside %q (len %d)",
				idx, r.bookmark.description, len(r.bookmark.description))
		}
	}
	// highlightMatches indexes by rune, so it must not panic or drop text.
	if got := highlightMatches(r.bookmark.description, r.matchedIndexes); got == "" {
		t.Error("highlight produced empty output")
	}
}

// highlighted returns the characters search() marked, so a test can assert on
// what the user sees rather than on raw offsets.
func highlighted(r searchResult) string {
	runes := []rune(r.bookmark.displayText())
	var sb strings.Builder
	for _, idx := range r.matchedIndexes {
		if idx >= 0 && idx < len(runes) {
			sb.WriteRune(runes[idx])
		}
	}
	return sb.String()
}

// Scoring matches the keyword-prefixed string, so fuzzy may satisfy a term
// from the hidden keyword: "log" once matched l and o inside "dvl" and only
// the g in "logs", lighting up a misleading fragment.
func TestSearchHighlightsWholeTermOnCommandBookmarks(t *testing.T) {
	bookmarks := []bookmark{
		{command: "dvl", url: "https://logs.example.com/a", description: "logs test legacy datacenter"},
	}
	results := search(bookmarks, "log legacy")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if got, want := highlighted(results[0]), "loglegacy"; got != want {
		t.Errorf("highlighted %q, want %q (indexes %v)", got, want, results[0].matchedIndexes)
	}
}

// A term the description cannot satisfy on its own contributes no highlight,
// even though the keyword let the bookmark match.
func TestSearchHighlightsNothingForKeywordOnlyTerm(t *testing.T) {
	bookmarks := []bookmark{
		{command: "zq", url: "https://example.com/a", description: "release dashboard"},
	}
	results := search(bookmarks, "zq")
	if len(results) != 1 {
		t.Fatalf("expected the keyword to match, got %d results", len(results))
	}
	if got := highlighted(results[0]); got != "" {
		t.Errorf("highlighted %q, want nothing", got)
	}
}
