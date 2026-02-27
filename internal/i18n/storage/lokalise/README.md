# Lokalise Storage Adapter

This package implements a `StorageAdapter` for `hyperlocalise`, backed by Lokalise API v2.

## Flow (ASCII)

```text
                          +------------------------+
                          | hyperlocalise syncsvc  |
                          +-----------+------------+
                                      |
                 +--------------------+--------------------+
                 |                                         |
              sync pull                                  sync push
          (remote -> local)                           (local -> remote)
                 |                                         |
                 v                                         v
      +------------------------+              +--------------------------+
      | Adapter.Pull()         |              | Adapter.Push()           |
      | - resolve locales      |              | - normalize entries      |
      | - map key/context/lang |              | - send Upsert payload    |
      +-----------+------------+              +------------+-------------+
                  |                                          |
                  v                                          v
      +------------------------+              +--------------------------+
      | HTTPClient.ListKeys()  |              | HTTPClient.Upsert...()   |
      | - Keys.List (paged)    |              | 1) Keys.List (paged)     |
      | - include translations |              | 2) split existing/missing|
      | - locale filtering     |              | 3) Keys.BulkUpdate       |
      +-----------+------------+              | 4) Keys.Create           |
                  |                           +------------+-------------+
                  v                                        |
      +------------------------+                           v
      | Lokalise API           |              +--------------------------+
      | /projects/{id}/keys    |<------------>| Lokalise API             |
      +------------------------+              | /projects/{id}/keys      |
                                              +--------------------------+
```

## Config

```jsonc
{
  "storage": {
    "adapter": "lokalise",
    "config": {
      "projectID": "your-project-id",
      "apiTokenEnv": "LOKALISE_API_TOKEN",
      "targetLanguages": ["fr", "de"],
      "timeoutSeconds": 30
    }
  }
}
```

Token must come from `LOKALISE_API_TOKEN` (or `apiTokenEnv` if customized).
