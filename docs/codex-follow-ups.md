# Codex follow-up messages

Sessions using the `codex` provider send follow-ups to the active run through
Codex app-server's `turn/steer` method. Messages sent while the server is starting
are held until the first turn starts, then submitted in order. If the turn has
already finished, the message starts a continuation on the same thread.

This requires a Codex CLI with app-server and `turn/steer` support (the protocol
was checked against 0.154.0). The Go SDK remains responsible for executable
discovery and event/input types; its `codex exec` streaming API does not expose
active-turn steering. The bidirectional transport lives in `internal/codexapp`.
No local SDK checkout or module replacement is required.

Existing Codex login credentials work without an API key. Explicit API keys use
a process-local provider configuration, without changing saved login credentials.
Text and attached local images are accepted as follow-ups. Model, permission,
and working-directory changes remain separate session operations.

A transport timeout or unknown steering error is reported instead of replaying
input whose acceptance is uncertain. Only an explicit “no active turn” rejection
is retried as a new turn. Direct OpenAI-compatible sessions are unaffected.

Regression checks:

```sh
go test ./internal/server/agents ./internal/codexapp
go test -race ./internal/codexapp ./internal/server/agents -run 'TestCodex|TestIsCodex|TestNotificationsDoNotBlockRPC|TestCancellationUnblocksCall'
```

Protocol reference: https://learn.chatgpt.com/docs/app-server#steer-an-active-turn
