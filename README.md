# bm — Bookmark Opener

A CLI tool for searching and opening bookmarks from the terminal.

## Install

```
go build -o bm .
```

Move the `bm` binary somewhere on your `$PATH`.

## Usage

```
bm [query...]        # fuzzy search bookmarks
bm <command> [arg]   # run a command shortcut
```

### Fuzzy search

```
$ bm wiki
1: Team wiki - https://wiki.example.com
2: Wikipedia - https://en.wikipedia.org

Where to go? 1
```

Multiple words are joined into a single query. If no query is given, you are prompted for one.

### Commands

Commands skip the search step and open a URL directly.

```
$ bm pr 42         # opens pull request #42 directly
$ bm pr            # prompts: "Enter id:"
$ bm dash          # opens directly (no parameter)
```

If the first argument doesn't match a command, it falls through to fuzzy search.

## ~/.bookmarks

Create a plain text file at `~/.bookmarks` with one bookmark per line.

### Format

```
<url> - <description>
```

- The first ` - ` separates the URL from the description
- Description is optional
- Lines starting with `#` are comments
- Blank lines are ignored

### Commands

Prefix a line with `!keyword` to define a command shortcut:

```
!keyword <url> - <description>
```

### Parameters

Use `{name}` in a URL as a placeholder. The name is shown in the prompt when no argument is provided.

### Example

```
!pr https://example.com/repo/pulls/{id} - Pull request
!dash https://example.com/dashboard - Dashboard

# Reference
https://example.com/docs - Documentation
https://example.com/search?q={query} - Search
```
