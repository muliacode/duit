# Security

duit is a fully local, single-user desktop app: no server, no account, and
no network call during normal operation. The threat model is therefore
about *this device*, not a network adversary; though a few network- and
supply-chain-related risks still apply.

## Data at rest

- The save file is a plain-text journal (see `docs/FILE_FORMAT.md`), written
  with `0600` permissions and stored wherever the user chooses via a native
  save dialog. duit never picks a location silently.
- Because the file is plain text, duit does **not** claim confidentiality
  against another local user or process with filesystem access to the same
  account. If you need that, put the file on an OS-encrypted volume
  (FileVault/BitLocker/LUKS) or encrypt it with something like `age` outside
  of duit. An in-app encrypted-file mode is tracked as a future option, not
  a v1 claim.
- Writes are atomic (temp file + `fsync` + rename), with rolling backups in
  a `.duit-backups/` directory next to the save file, so a crash or power
  loss mid-save should not corrupt or silently truncate the user's data.

## No telemetry, no network

- duit's Wails asset server only serves its own embedded, locally built
  frontend bundle; it does not fetch anything over HTTP at runtime.
- No analytics, crash reporting, or update-check network call is included.
  If one is added later, it must be opt-in and disclosed here.

## Webview hardening

- The Wails webview's default context menu and DevTools are disabled in
  production builds.
- A strict Content-Security-Policy is set on the asset server
  (`default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`
  for Tailwind's injected styles), so an injected script has nowhere to
  reach out to, even though the app never loads remote content in the
  first place.
- The frontend never uses `v-html`/`innerHTML` with data derived from the
  journal file. Payee/tag text is user-controlled and rendered as text
  nodes only, which rules out stored XSS via the save file.

## Supply chain

- Go and npm dependencies are pinned (`go.sum`, `package-lock.json`) and
  should be checked with `govulncheck` and `npm audit` in CI before release.

## Reporting

Contact `noah@glorycode.nl` for security-related findings.