# bookmark

A CLI for searching and opening bookmarks. `bm` reads `~/.bookmarks`.

## Never put real bookmark data in this repo

`~/.bookmarks` is personal and names internal systems. Nothing from it belongs
in tracked files, commit messages, or the README — not descriptions, not URLs,
and not `!keyword` shortcuts.

This has gone wrong twice by building test fixtures from the real file, so:

- Invent fixture data. Use `example.com` hosts and descriptions that name
  nothing real.
- Keywords in fixtures are invented too (`dv`, `dvl`, `pv`, `panel`), not
  copied from `~/.bookmarks`.
- Reading `~/.bookmarks` to try the tool or debug is fine. Pasting what you
  find into a file, a test, or a commit message is not.
- Before committing, check the staged diff. Individual words like `logs`,
  `prod` and `cloud` are generic and appear in invented fixtures too, so
  matching on those only produces noise. The real failure mode is copying a
  whole description or a keyword, so grep for those:

  ```sh
  # whole descriptions, verbatim
  git diff --cached | grep -E "^\+" |
    grep -Ff <(sed -n 's/.* - //p' ~/.bookmarks | awk 'length>8')

  # keywords, only where a fixture declares one
  git diff --cached | grep -E '^\+.*command: "' |
    grep -Ef <(sed -n 's/^!\([a-zA-Z0-9-]*\).*/command: "\1"/p' ~/.bookmarks)
  ```

  Both should print nothing. Prose counts too — the leak that slipped through
  last time was a keyword inside a doc comment, not in a fixture.

Some real data predates this rule, in the `TestSearchMatchesOnDescriptionOnly`
fixture and in the pushed history. Leave it; do not rewrite history for it.

## Build and test

```sh
go test ./...
go build -o bm .   # `bm` is gitignored; run ./bm to avoid the Homebrew copy
```

`go build ./...` drops a stray `bookmark` binary in the repo root — it is not
gitignored. Use `go build -o /dev/null ./...` for a compile check.

## Releasing

Tag `v*` and push the tag; `.github/workflows/release.yml` runs GoReleaser,
which publishes the GitHub release and updates the `martensjostrand/homebrew-tap`
formula. The formula is named `bookmark`; the binary is `bm`.
