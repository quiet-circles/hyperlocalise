# Crowdin storage adapter

Crowdin adapter for `hyperlocalise sync` operations.

## Config

```json
{
  "adapter": "crowdin",
  "config": {
    "projectID": "123456",
    "apiTokenEnv": "CROWDIN_API_TOKEN",
    "sourceLanguage": "en",
    "targetLanguages": ["fr", "de"]
  }
}
```

Set `CROWDIN_API_TOKEN` in your environment.
`projectID` must be the numeric Crowdin project ID.
