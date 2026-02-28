# Smartling Storage Adapter

This package implements a `StorageAdapter` for `hyperlocalise` using Smartling APIs.

## Config

```jsonc
{
  "storage": {
    "adapter": "smartling",
    "config": {
      "projectID": "your-project-id",
      "userIdentifier": "your-user-identifier",
      "userSecretEnv": "SMARTLING_USER_SECRET",
      "targetLanguages": ["fr", "de"],
      "timeoutSeconds": 30
    }
  }
}
```

Token/secret must come from `SMARTLING_USER_SECRET` (or `userSecretEnv` if customized).
