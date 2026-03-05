# translationfileparser

`translationfileparser` provides a strategy-based parser layer for local translation files.

## Supported formats

- `.json` via `JSONParser`
- `.arb` via `ARBParser` (Flutter Application Resource Bundle)
- `.xlf` / `.xliff` via `XLIFFParser` (XLIFF 1.2 and 2.x)
- `.po` via `POFileParser` (GNU gettext)
- `.md` / `.mdx` via `MarkdownParser`
- `.strings` via `AppleStringsParser` (Apple/Xcode strings files)
- `.stringsdict` via `AppleStringsdictParser` (Apple/Xcode plural dictionaries)
- `.xcstrings` via `XCStringsParser` (Apple/Xcode Strings Catalog)
- `.csv` via `CSVParser` (key/value and per-locale column layouts)

## Strategy API

- `NewDefaultStrategy()` returns a strategy pre-registered with all supported parsers listed above.
- `Register(ext, parser)` allows adding/replacing parser implementations by extension.
- `Parse(path, content)` resolves parser by extension and returns `map[string]string`.
- `ParseForLocale(path, content, locale)` resolves parser by extension and returns values for a preferred locale when supported by that parser (otherwise falls back to `Parse`).

## Parser behavior

### JSON

- Accepts object-shaped JSON.
- Nested objects are flattened with dotted keys.
  - Example: `{ "home": { "title": "Accueil" } }` -> `home.title=Accueil`
- Non-string leaf values are rejected.

### ARB

- Accepts object-shaped ARB JSON (Flutter resource bundles).
- Only top-level non-metadata keys are treated as translatable message entries.
- Keys prefixed with `@` (for example `@hello`, `@@locale`) are treated as metadata and excluded from translation parsing.
- `MarshalARB(template, values)` preserves ARB metadata keys and rewrites only message values.

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

### Apple Strings (`.strings`)

- Parses `"key" = "value";` entries into `map[string]string`.
- Ignores line comments (`// ...`) and block comments (`/* ... */`).
- Decodes escaped sequences (`\n`, `\r`, `\t`, `\"`, `\\`) and unicode escapes (`\u`, `\Uhhhh`, surrogate pairs).
- Supports multiline quoted value content.
- `MarshalAppleStrings(template, values)` preserves template layout/comments/spacing and replaces only value literals.

### Apple Stringsdict (`.stringsdict`)

- Parses plist/XML dictionaries and flattens `<string>` leaves to dotted keys.
  - Example: `item_count.items.one=%d item`
- Preserves plural category keys (`zero`, `one`, `two`, `few`, `many`, `other`) as part of flattened key paths.
- `MarshalAppleStringsdict(template, values)` preserves plist/XML layout and replaces only `<string>` text values.

### Apple Strings Catalog (`.xcstrings`)

- Parses `strings.*.localizations.*` `stringUnit.value` leaves into `map[string]string`.
- Flattens variation branches using dotted keys.
  - Example: `item_count.plural.one=1 item`
- Preserves catalog metadata/state fields when marshalling.
- `MarshalXCStrings(template, values, targetLocale)` updates only targeted localized values while preserving structure.
- Invariant: if catalog-level `sourceLanguage` is set, each entry being parsed/cloned must contain that locale under `localizations`; otherwise parsing/marshalling returns an error.
- Fallback rule: only when `sourceLanguage` is empty, parser selection falls back deterministically to the first locale key (sorted order).

## Minimal usage

```go
strategy := translationfileparser.NewDefaultStrategy()

values, err := strategy.Parse("lang/fr.xliff", content)
if err != nil {
    return err
}

fmt.Println(values["checkout.submit"])
```
