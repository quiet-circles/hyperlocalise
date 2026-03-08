# translationfileparser

`translationfileparser` provides a strategy-based parser layer for local translation files.

## Supported formats

- `.json` via `JSONParser`
- `.jsonc` via `JSONCParser`
- `.arb` via `ARBParser` (Flutter Application Resource Bundle)
- `.xlf` / `.xliff` via `XLIFFParser` (XLIFF 1.2 and 2.x)
- `.po` via `POFileParser` (GNU gettext)
- `.md` / `.mdx` via `MarkdownParser`
- `.strings` via `AppleStringsParser` (Apple/Xcode strings files)
- `.stringsdict` via `AppleStringsdictParser` (Apple/Xcode plural dictionaries)
- `.csv` via `CSVParser` (key/value and per-locale column layouts)

## Strategy API

- `NewDefaultStrategy()` returns a strategy pre-registered with JSON, XLIFF, PO, Apple Strings and Markdown/MDX parsers.
- `Register(ext, parser)` allows adding/replacing parser implementations by extension.
- `Parse(path, content)` resolves parser by extension and returns `map[string]string`.

## Parser behavior

### JSON

- Accepts object-shaped JSON.
- Nested objects are flattened with dotted keys.
  - Example: `{ "home": { "title": "Accueil" } }` -> `home.title=Accueil`
- Non-string leaf values are rejected.

### JSONC

- Accepts JSON with `//` and `/* ... */` comments plus trailing commas.
- Produces the same flattened dotted-key output shape as the JSON parser.
- Non-string leaf values are rejected.

### ARB

- Accepts object-shaped ARB JSON (Flutter resource bundles).
- Only top-level non-metadata keys are treated as translatable message entries.
- Keys prefixed with `@` (for example `@hello`, `@@locale`) are treated as metadata and excluded from translation parsing.
- `MarshalARB(template, sourceTemplate, values, targetLocale)` preserves target-template metadata and ordering, carries source `@key` metadata forward for newly appended message keys, and normalizes `@@locale` to `targetLocale`.

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
- Treats `NSString*` fields such as `NSStringLocalizedFormatKey` and `NSStringFormatSpecTypeKey` as structural metadata, not translatable content.
- Validates that every `%#@token@` in `NSStringLocalizedFormatKey` matches a sibling substitution dictionary key.
- Preserves plural category keys (`zero`, `one`, `two`, `few`, `many`, `other`) as part of flattened key paths.
- `MarshalAppleStringsdict(template, values)` preserves plist/XML layout and replaces only `<string>` text values.

## Minimal usage

```go
strategy := translationfileparser.NewDefaultStrategy()

values, err := strategy.Parse("lang/fr.xliff", content)
if err != nil {
    return err
}

fmt.Println(values["checkout.submit"])
```
