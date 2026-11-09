# Security

Read the [project disclaimer](DISCLAIMER.txt). Testing is not a security audit
or a guarantee that vulnerabilities are absent.

## Intended boundary

- LuCI handles login. The CGI verifies the `network-speed-test/test/run` session
  permission; it cannot mint a grant for an unauthorised session.
- The HTTP service accepts a cryptographically random 256-bit test token, tied
  to the client IP and expiry. It cannot use an admin session as a native token.
  Starting a run revalidates the original LuCI permission. Operations are fixed;
  there is no arbitrary command, file-upload or router-administration API.
- Cross-origin responses allow only the exact origin that prepared the current
  client-bound grant. No wildcard origin or credentialed CORS is enabled.
  Preflight alone grants no permission to transfer data. Same-origin native
  pages and non-browser clients still need a valid token.
- Listener sockets are bound to selected LAN/VPN devices. Automatic VPN
  discovery follows existing trusted firewall-zone input policy. No firewall,
  web-server TLS, or VPN configuration is modified. See the precise discovery
  limits and overrides in [architecture](docs/ARCHITECTURE.md).
- Uploads are discarded; samples/results live in RAM. Request sizes, deadlines,
  exclusive file locks and a run lease bound normal usage. These are not a
  comprehensive denial-of-service defence. The service runs as root to use
  interface binding and router metadata/session APIs; a server bug could have
  serious consequences.

HTTP test traffic is deliberately unencrypted to reduce router CPU use. Someone
able to observe or manipulate that path may capture a temporary token or alter
measurements. Tokens are scoped, but HTTP is not suitable for hostile networks.
Over an encrypted VPN, traffic inside the tunnel benefits from that VPN's
protection; the test still includes its encryption/decryption overhead.

Keep LuCI on HTTPS and do not publish port 18080 to the internet. Do not disable
browser protections to bypass a failed test. Trust only appropriate APK signing
keys: installing this release's public key authorises APKs signed by that key.

## Reporting

Use GitHub's **Security → Report a vulnerability** for a private report:
[report privately](https://github.com/audrius-paskevicius/openwrt-network-speed-test/security/advisories/new).
Include the affected version, a minimal reproduction and expected/actual
behaviour. Do not put exploit details or credentials in public issues.
No response time, maintenance or fix is promised; see the disclaimer.

Remove passwords, private keys, cookies, tokens, client addresses and device
identifiers from reports. Never attach `grant.json`, `run.json`, browser profiles,
HAR captures or full router configuration without reviewing and sanitising them.
Only the current published release is the verification baseline; there is no
commitment to backport fixes.
