# Security Policy

## Supported versions

Security fixes are applied to the latest release line. Older releases do not receive backported fixes; upgrade to the latest version to stay patched.

## Reporting a vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Report them privately through one of these channels:

- GitHub private vulnerability reporting: use the "Report a vulnerability" button under the repository's Security tab (https://github.com/clouddrove/smurf/security/advisories/new).
- Email: hello@clouddrove.com with the subject line "smurf security report".

Include the affected version, reproduction steps, and the impact you believe the issue has. You should receive an acknowledgment within a few business days.

## Scope notes for users

- `smurf.yaml` can hold registry credentials as a fallback. Prefer `${ENV_VAR}` interpolation over literal secrets, never commit the file with credentials in it, and note that smurf creates it with 0600 permissions.
- Terraform state and its backups routinely contain secrets; smurf writes state backups with 0600 permissions.
- The optional AI error explanation (`--ai`) sends redacted error text to the OpenAI API. Common token formats are masked before sending, but review `internal/ai/redact.go` for the exact patterns if this matters in your environment.
