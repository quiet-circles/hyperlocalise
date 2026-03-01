# translationfileparser

`translationfileparser` provides a strategy-based parser layer for local translation files.

## Supported formats

- `.json` via `JSONParser`
- `.xlf` / `.xliff` via `XLIFFParser` (XLIFF 1.2 and 2.x)
- `.po` via `POFileParser` (GNU gettext)
- `.md` / `.mdx` via `MarkdownParser`

## Strategy API

- `NewDefaultStrategy()` returns a strategy pre-registered with JSON, XLIFF, PO, and Markdown/MDX parsers.
- `Register(ext, parser)` allows adding/replacing parser implementations by extension.
- `Parse(path, content)` resolves parser by extension and returns `map[string]string`.

## Parser behavior

### JSON

- Accepts object-shaped JSON.
- Nested objects are flattened with dotted keys.
  - Example: `{ "home": { "title": "Accueil" } }` -> `home.title=Accueil`
- Non-string leaf values are rejected.

### XLIFF

- Reads keys from `id` first, then `name`, then `resname`.
- Supports `<trans-unit>` (1.2) and `<unit>` (2.x).
- Uses `<target>` when present, falls back to `<source>` when target is empty.

### PO

- Reads `msgid` -> `msgstr` mappings.
- Supports multiline quoted continuations.
- For plural forms, uses `msgstr[0]` as the mapped value.
- Skips header entry (`msgid ""`).
- Ignores comments and `msgctxt` for now.

### Markdown

- Extracts stable sequential keys (`md.0001`, `md.0002`, ...).
- Preserves frontmatter blocks (`---`) unchanged.
- Preserves fenced code blocks (``` and ~~~) unchanged.
- Preserves Markdown syntax tokens and link destinations while extracting text segments.

## Minimal usage

```go
strategy := translationfileparser.NewDefaultStrategy()

values, err := strategy.Parse("lang/fr.xliff", content)
if err != nil {
    return err
}

fmt.Println(values["checkout.submit"])
```
