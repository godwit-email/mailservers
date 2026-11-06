# mailservers

Finds the IMAP and SMTP servers of an email domain: [Thunderbird's ISPDB](https://github.com/thunderbird/autoconfig), kept in sync, plus what it lacks, above all hosted corporate mail in China (Coremail, Alibaba, NetEase, 263, Sina) matched by MX record.

The package makes no network requests. You look up the MX record and pass it in, so no address or domain leaves the machine because of this package.

```go
import "github.com/godwit-email/mailservers"

p, ok := mailservers.ByDomain("aol.de")
// imap.aol.com:993

p, ok = mailservers.ByMX("corp.example", "mx.c3.icoremail.net.")
// c3.icoremail.net:993

username := mailservers.Login(p.IMAP[0], "you@corp.example")
```

`IMAP` and `SMTP` list a provider's servers in order of preference. Check `Auth` before offering a password: some servers, such as Outlook's, accept only `OAuth2`. `Enable`, if set, says what the user must turn on first.

## Two sizes

| | Contents | Size (gzipped) |
|---|---|---|
| `mailservers` (`min.json`) | Encrypted IMAP and SMTP servers, sign-in methods, help and enable links | about 13 KB |
| `mailservers/full` (`full/full.json`) | Also plaintext servers, POP3, Exchange/OWA/Graph, OAuth endpoints, webmail and documentation links | about 20 KB |

Go links only what you import, so using the root package does not carry the full data. For other languages, each [release](https://github.com/godwit-email/mailservers/releases) attaches both JSON files. For the untouched XML, see the `upstream` submodule.

The full data leaves out webmail form selectors and Thunderbird's per-server preferences. Do not offer its plaintext servers to users without saying so.

## How the data is built

```
upstream/   ISPDB, as a git submodule
overlay/    our entries, in ISPDB's format
   | go run ./cmd/gen
min.json, full/full.json
```

An overlay file wins over an upstream file of the same name, and over any upstream provider for each domain it lists. It may add `<mx>` rules, which ISPDB does not have:

- `qiye.163.com` matches that host and its subdomains.
- `hzmx*.mxmail.netease.com` matches label by label; `*` stays within one label.
- `*.{cluster}.icoremail.net` captures a label and fills `{cluster}` in the servers.

The rule with the most labels wins. Add an entry only after checking that its hosts present a valid TLS certificate and accept a login, and send what fits upstream to ISPDB too.

Module consumers do not get submodule contents, so the generated files are committed. Run `git clone --recurse-submodules` to work on them.

## Sync

Every week `sync.yml` moves the submodule, regenerates and tests. Because this data decides where passwords go:

- If providers or addresses were only added, it commits to `main` and publishes a release.
- If anything an existing address relies on changed or disappeared, it opens a `needs-review` pull request listing each change. Merging that publishes a release.

Versions look like `v0.20261106.12`: date and run number.

## License

Code: Apache-2.0. Data derived from ISPDB: MPL-2.0. See [NOTICE](NOTICE).
