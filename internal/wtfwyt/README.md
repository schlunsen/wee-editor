# wtfwyt export

Detects credentials in session content and reports them to a
[wtfwyt](https://wtfwyt.com) server.

Answers one question: **did a secret end up in a chat transcript?**

## Secrets never leave the machine

Detection runs here, in the client, before anything is transmitted:

1. Content is scanned with 28 rules plus entropy-gated generic detection.
2. Every credential found is replaced with `[REDACTED:rule:fingerprint]`.
3. Only a *finding* is sent — rule, severity, and a truncated SHA-256 of the
   value.

```
on disk:      GITHUB_TOKEN=ghp_u8jzPde0IgxLd6GncfBAepfJBd0Kh8oOOL8d
transmitted:  GITHUB_TOKEN=[REDACTED:github_token:faeb1f0959e2c7f9]
```

This ordering is the entire design. Shipping raw transcripts to a server and
scanning them *there* would turn that server into a durable central store of
every secret anyone pasted into a chat — a far worse exposure than the one
being reported. The fingerprint is enough to say *this same key leaked in six
sessions* and useless to anyone who obtains it.

## Configuration

In `~/.claude/wee-config.json`:

```json
{
  "wtfwyt": {
    "enabled": true,
    "endpoint": "https://wtfwyt.com",
    "client_id": "wee_...",
    "export_content": false,
    "alert_on_critical": true
  }
}
```

The client secret is **never** written to the config file. It is read from the
environment, following the same convention as `TunnelSettings.AuthToken`:

```bash
export WTFWYT_CLIENT_SECRET='...'
```

Get credentials from the server operator:

```bash
wtfwytd create-client -email you@example.com -name "my laptop"
```

### `export_content`

| Value | What is transmitted |
|---|---|
| `false` *(default)* | Findings only — fingerprint, rule, and where it surfaced. No transcript content of any kind. |
| `true` | Findings plus message and tool content, with every credential replaced by a placeholder. |

The default is deliberately the conservative one. Turning on content export is
an explicit decision.

## Where it hooks in

`ConversationParser.recordToolExecution` — the single point every completed
tool execution passes through. Tool results are where credentials actually
surface: nobody types their AWS key into a chat, they run `cat .env`,
`env | grep`, or `kubectl get secret -o yaml`, and the output lands in the
transcript, and from there in the model provider's logs.

Assistant *thinking* blocks are scanned too. A model reasoning about a key it
just read will quote it back.

The hook is an interface defined in `internal/analytics`, so that package does
not import this one and there is no dependency cycle. When export is disabled
the hook is never installed and parsing behaves exactly as before.

## Local alerts

A detection prints to the terminal the moment it happens, before any network
call:

```
✗ credential exposed: GitHub token (ghp_****8d) in tool_result (Bash) - rotate it
```

Whoever just exposed a live credential needs to know now, not when someone
next opens a dashboard.

## If the server reports `unredacted`

The server re-runs the same rules on what arrives. It should find nothing. If
it does, this client's rules are older than the server's and it is
transmitting raw credentials — the exporter logs a warning saying so. Upgrade.

## Delivery

Events are batched (5s or 200 events), retried with exponential backoff and
jitter, and flushed once more on shutdown. Every event carries a stable
client-generated id and the server deduplicates on it, so a retry after an
ambiguous timeout cannot double-write.

The send buffer is bounded (10k events). When it is full the **oldest** event
is dropped, matching `logging.LogBroadcaster`: recent context is worth more
than a complete but stale backlog, and blocking would stall the editor.
