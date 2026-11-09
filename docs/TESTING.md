# Verification

## 0.1.0-r3 — initial implementation, 2026-11-09

- Go unit tests and vet passed. All nine backend tests also passed natively on
  the OpenWrt ARMv7 router, including the actual Linux file-lock test.
- Browser checks passed for the single Start control, automatic timing, confirmed
  upload byte counts, authenticated requests, mobile/desktop layout, expired login,
  busy tests, partial uploads, duplicate starts and hidden-page interruption.
- Disclaimer checks cover canonical text, first use, persisted acceptance across
  fresh modules, changed wording, blocked storage and failed/empty text loads.
- The native LuCI page was exercised with a normal router login. The existing
  session was reused for all test operations. First-use acceptance survived a page
  reload; the permanent disclaimer tab was accessible without another prompt.
- A real browser-to-router download/upload/latency run completed on a 2.4 GHz
  client connection. This verifies the actual local path and byte flow, not
  maximum radio capacity or equivalence to an independently calibrated benchmark.
- Unauthenticated access was rejected by the LuCI page and all five CGI
  operations. Network, Wi-Fi, firewall, DHCP, web-server and existing application
  configuration hashes were unchanged by installation.
- The signed package is checked against its public key and all 14 payload files
  are compared byte for byte against intended sources/generated content. The r3
  revision refines the native layout and text encoding; the tested backend
  binary is unchanged.

No internet speed-test server is used. Browser-fixture results are synthetic and
must not be interpreted as measured network performance.
