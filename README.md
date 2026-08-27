# bm — Bookmark Opener

A CLI tool for searching and opening bookmarks from the terminal.

## Install

```
go build -o bm .
```

Move the `bm` binary somewhere on your `$PATH`.

## Usage

```
bm [query...]    # open the search list, optionally prefilled
```

Results filter as you type. Word order does not matter — `wiki team` finds the
same bookmarks as `team wiki`.

```
$ bm wiki
Search: wiki
▸  1 Team wiki
      https://wiki.example.com
   2 Wikipedia
  1/2   ↑↓ move · enter open · esc quit
```

| Key | |
|---|---|
| `↑` `↓`, `ctrl-p` `ctrl-n` | move |
| `enter` | open, or prompt for a `{parameter}` |
| `esc`, `ctrl-c` | quit |

`bm` needs an interactive terminal; it does not read from a pipe.

### Commands

A `!keyword` ranks its own bookmark first, so `bm dash` puts the dashboard at
the top — but you still confirm with `enter` and can pick something else. Any
argument after the keyword is ignored: `bm pr 42` searches for `pr`, and `42`
is typed at the parameter prompt.

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

Use `{name}` in a URL as a placeholder. The name is shown in the prompt after
you select the bookmark.

### Example

```
!pr https://example.com/repo/pulls/{id} - Pull request
!dash https://example.com/dashboard - Dashboard

# Reference
https://example.com/docs - Documentation
https://example.com/search?q={query} - Search
```
