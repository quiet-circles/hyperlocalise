# POEditor Storage Adapter

This package implements the first `StorageAdapter` for `hyperlocalise`, backed by POEditor.

It maps POEditor terms and translations into the normalized sync model used by `sync pull` / `sync push`.

For the shared storage model, provenance metadata, and LLM -> curation flow, see:

- [`../README.md`](../README.md)

## What it does

- Pull curated translations from POEditor into normalized entries
- Push local normalized entries to POEditor
- Map POEditor `term + context + language` to local entry identity:
  - `Key` = POEditor term
  - `Context` = POEditor context
  - `Locale` = POEditor language

## Auth (v1)

POEditor auth is API token only.

Token resolution order:

1. env var from `storage.config.apiTokenEnv`
2. `POEDITOR_API_TOKEN`

## Adapter config (used in `i18n.jsonc`)

```jsonc
{
  "storage": {
    "adapter": "poeditor",
    "config": {
      "projectID": "123456",
      "apiTokenEnv": "POEDITOR_API_TOKEN"
    }
  }
}
```

Supported config fields:

- `projectID` (required)
- `apiTokenEnv` (optional, defaults to `POEDITOR_API_TOKEN`)
- `sourceLanguage` (optional)
- `targetLanguages` (optional)
- `timeoutSeconds` (optional)

## Behavior notes (v1)

- Empty translation values are skipped on pull/push
- Namespaces are local-only (POEditor adapter does not map namespaces)
- Tags/comments/plurals are not part of v1 sync semantics
- Conflict resolution is handled by `syncsvc`, not by this adapter

## Testing strategy

The adapter is built around a small `Client` interface:

- `ListTerms(...)`
- `UpsertTranslations(...)`

Use `NewWithClient(...)` in tests to inject a fake client and validate mapping behavior without network access.
