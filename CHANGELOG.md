# Changelog

## 0.2.2-r1 — 2026-11-09

First public release.

- Automatically serve tests on LAN and recognised VPN interfaces already trusted
  for router input by the firewall; keep sockets bound to selected devices.
- Show the VPN path instead of suggesting a local Wi-Fi radio measurement.
  Add configuration overrides for unusual VPNs and exclusions, without another
  UI tab or changes to firewall rules.
- Keep the HTTPS LuCI UI open while using scoped, direct HTTP test transfers.
- Include reviewed installation, architecture, security, release and validation
  documentation. Publish a signed ARMv7 APK, source archive and checksums.

## 0.2.1-r1 — 2026-11-09 (local development)

- Replace HTTPS-to-HTTP page navigation with in-place LuCI tests.
- Restrict cross-origin HTTP access to the authenticated issuing origin and
  client-bound test grant. Preserve CORS headers on recycled upload sockets.

## 0.2.0-r2 — 2026-11-09 (local development)

- Reuse upload TCP connections, enlarge the client-generated sample to 1 MiB,
  and adapt upload blocks up to 64 MiB to reduce per-block overhead.

## 0.2.0-r1 — 2026-11-09 (local development)

- Introduce a LAN-bound native HTTP transfer service with scoped tokens,
  client-generated samples, Linux sendfile/splice and low router CPU overhead.

## 0.1.x — 2026-11-09 (local development)

- Initial LuCI test and persistent disclaimer, compact controls, live radio
  identification and latest-result diagnostics in RAM.

See [verification history](docs/TESTING.md) for measurements and test scope.
