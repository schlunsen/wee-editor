# Security

Please report suspected vulnerabilities privately. Do not include credentials, session exports, personal data, or exploit details in public issues.

Use [GitHub private vulnerability reporting](https://github.com/schlunsen/wee-editor/security/advisories/new) when available. If GitHub does not offer that option, open an issue requesting a private contact channel without disclosing the vulnerability.

Include the affected version, reproduction steps using synthetic data, expected and actual behavior, and the potential impact. Remove secrets from logs and screenshots before sharing them.

Security fixes target the latest release. Older releases may require an upgrade.

## Secret checks

The Secret Scan workflow runs Gitleaks against the tracked source snapshot on pushes and pull requests. Findings fail the workflow; output is redacted. It does not scan old commits, GitHub comments, workflow logs, release binaries, or image contents.

Before publishing repository history, also run `gitleaks git . --log-opts="--all" --redact` and review each finding. If a credential is real, revoke or rotate it before deciding whether history cleanup is needed. Deleting a file from the latest commit does not remove older copies.
