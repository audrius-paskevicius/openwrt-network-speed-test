# Verification

This is versioned evidence, not a description of every current behaviour.
Older sections describe superseded workflows and measurements. The current
implementation is described in [README](../README.md) and
[architecture](ARCHITECTURE.md). Throughput observations are examples, not
performance guarantees; test identifiers in source are synthetic fixtures.

## Post-release verification and documentation review, 2026-11-09

- The user completed an external WireGuard test and confirmed the
  `WireGuard VPN` label. The saved result independently showed that path at
  both start and finish, with no connection change: 56.1 Mbps download,
  11.3 Mbps upload and 99.8 ms HTTP latency. The busiest router core averaged
  13% busy. These values describe that VPN path and run, not maximum capacity.
- This confirms one real external WireGuard path using a local hostname
  resolving to the router's private address. Other VPN protocols have discovery
  fixture coverage, not equivalent end-to-end verification. The exact external
  client/browser version was not recorded.
- Reviewed the README, architecture, security, build/release instructions and
  changelog against the current code. Clarified signing-key verification,
  exact-match VPN discovery, connection-change detection and the difference
  between a versioned release snapshot and later documentation updates.
- Public evidence includes only rounded measurements and path type; it omits
  client addresses, private interface names, credentials and session identifiers.

## 0.2.2-r1 - VPN support and first public release, 2026-11-09

- Added automatic discovery of trusted VPN interfaces, with individual
  device-bound listeners, shared global run locks, and VPN path identification.
  Discovery fixtures cover trusted/untrusted tunnels, TUN, WAN devices/zones,
  manual overrides, exclusions, down interfaces and disabled automatic discovery.

- Reviewed documentation against the control flow, authentication, measurement
  timing, RAM files, lifecycle scripts and APK contents. Corrected the timing
  target and best-effort result-saving description; clarified compatibility
  endpoints, uninstall leftovers, signing-key trust and browser requirements.
- Added installation/troubleshooting, security, architecture, contribution and
  release instructions, a changelog, and automated documentation consistency
  checks. Measurement arithmetic and authentication are unchanged from 0.2.1-r1.
- After 0.2.1-r1 installation, the user reported slightly better local results
  and a pleasant UI. No new numerical result or exact browser version was supplied.
- Publication checks include backend tests/vet, JS and documentation checks,
  synthetic browser/disclaimer tests, native ARM tests, signed APK verification,
  exact package-content comparison and scans of Git history and release files.
  CI results are available in the repository Actions page.
- All 22 backend tests passed on Windows and natively on ARM; Go vet, JS checks,
  browser/disclaimer checks and documentation checks passed. Discovery fixtures
  passed in the router's BusyBox shell. The signed APK's 26 files matched the
  intended source/generated contents with no unexpected payload files.
- Automatic discovery on the target router selected LAN and its existing
  WireGuard interface; both procd instances were running after installation.
  Network, firewall and uhttpd configuration hashes were unchanged.
- Two local HTTPS LuCI runs completed at 44.4/43.4 Mbps with 8.8 ms latency and
  44.7/37.6 Mbps with 8.0 ms latency. The 48 upload responses used two TCP
  connections total. No admin session or cookies reached the HTTP endpoint;
  navigation, reload and mobile/desktop layout checks passed.
- Gitleaks found no secrets in the development history or extracted APK.
  Additional checks found no private-key blocks, personal filesystem paths,
  known deployment password or active test-session identifier in reachable Git
  objects or the extracted package. These checks are not proof against every
  possible form of sensitive data.

## 0.2.1-r1 - HTTPS LuCI with direct HTTP transfers, 2026-11-09

- Start stays in the current LuCI page. Each run obtains a fresh scoped token
  over the LuCI connection, then sends the test data directly to LAN port 18080.
  Idle radio lookup continues using the original LuCI connection and session.
- Cross-origin access is restricted to the exact origin stored by authenticated
  preparation, the same client IP, and an unexpired grant. Preflight permits
  only POST with Authorization/Content-Type; it grants no test access itself.
  Actual operations still validate the token, and Start rechecks LuCI access.
  Upload responses preserve these headers through the kernel splice path.
- Go tests and vet, JS syntax checks, browser fixtures and disclaimer checks
  passed. All 21 tests passed natively on ARM. Tests cover wrong origin/client,
  expired grants, admin session rejection on the native API, revoked sessions,
  cross-origin downloads/uploads, fresh preparation on retry, browser blocking,
  and separation of control credentials from HTTP transfer tokens.
- Desktop Chrome 157 completed two consecutive real runs without navigation:
  radio1, 2.4 GHz, channel 13; 40.6/38.0 Mbps and 8.3 ms, then 40.8/36.9 Mbps
  and 8.8 ms. All 40 upload responses used two TCP connections total. These
  figures reflect this PC's limited Wi-Fi adapter, not the phone's faster radio.
- Browser network events confirmed direct HTTP payloads, exact HTTPS CORS origin,
  and no LuCI session token or cookies in HTTP requests. No mixed-content bypass
  flags or browser permission overrides were used; only the existing router's
  self-signed certificate was accepted in the disposable test browser.
  Reload did not auto-start; 390/1280-pixel layouts were checked. The user later
  reported successful use; the exact client/browser version was not recorded.
- An initial second-run automation timeout was resolved by limiting DevTools
  request-body capture; the subsequent two-run check passed in full.
- Signed APK verification and comparison of all 19 payload files passed.
  Installed service rejects unauthorised requests and unrelated-origin
  preflights. Network, firewall and uhttpd configuration checksums are unchanged.

## 0.2.0-r2 - retain TCP connections and reduce block overhead

- Native uploads now return their hijacked sockets to the HTTP server after
  confirmation instead of closing TCP after every block. Authentication still
  applies to every request. Tests cover repeated uploads, pipelined bytes,
  invalid tokens on reused sockets, and listener shutdown.
- The browser-generated download sample increased from 64 KiB to 1 MiB, reducing
  file seek/send iterations. Uploads adapt towards 1.25-second blocks, bounded
  to 64 MiB. The server also accepts the older 64 KiB sample for existing pages.
- All 20 native ARM tests passed. The connection reuse/pipelining test passed
  30 repetitions. Router tracing again confirmed sendfile64 and splice without
  syscall errors. The signed APK's 19 files matched intended contents.
- A real browser completed 10 upload requests using one TCP connection, retained
  the HTTPS LuCI-to-HTTP scoped-token handoff, and exposed no admin session or
  cookies to the data API. Mobile layout and no automatic rerun on reload passed.
- The phone's radio0/channel-128 run measured 370.22 Mbps download, 490.46 Mbps
  upload and 5.3 ms latency, without roaming. The busiest core averaged 45% busy;
  its soft-interrupt time was 43%. Upload rose from 390.53 Mbps; download was
  essentially unchanged from 371.52 Mbps in the preceding version.
- The user then measured Google Search's Hamburg test at 388/462 Mbps and 27 ms,
  versus their previous 413/419 Mbps. The latest local result is about 4.6% lower
  downstream and 6.2% higher upstream. Separate runs vary; these results do not
  demonstrate a fixed remaining deficit or prove a maximum radio capacity.
- Network, uhttpd and firewall configuration hashes remained unchanged.

## 0.2.0-r1 - client-generated data and direct kernel transfers

- The phone's prior recorded HTTPS test was 52.5/50.3 Mbps on radio0, 5 GHz,
  channel 128, without roaming. One core averaged 92% busy (89% user time).
- Bulk transfers now use a LAN-device-bound native HTTP service. LuCI mints a
  30-minute, client-bound test capability. The browser navigates with this token
  in a fragment, removes the fragment immediately, and never passes the LuCI
  session to the data service. Each new run revalidates the original login.
- The client generates the 64 KiB reusable sample and upload buffers, chooses
  measurement duration, measures received/confirmed bytes and renders results.
  The router retains the sample only in RAM and discards all upload data.
- All 18 ARM backend tests passed, including real TCP transfer/byte checks,
  scoped-token isolation, client binding, revocation, expiry, truncated sample
  handling and Linux locking. The transfer/revocation cycle passed 30 repeats
  after fixing release of the upload lock before acknowledging its bytes.
- On-router syscall tracing confirmed sendfile64 for the download and splice
  for the upload, with no syscall errors. These are real kernel paths, not only
  an assumption based on the Go source. Unsupported systems retain Go fallbacks.
- A real browser started on the HTTPS LuCI page and ran the HTTP test with one
  click. Captured requests verified no LuCI session identifier or cookies went
  to the data API. A reload did not automatically repeat the test; mobile and
  desktop layouts fit their viewports and navigation returned to HTTPS.
- That development computer (72 Mbps Wi-Fi link, radio1, channel 13) measured
  42.7/38.9 Mbps and 5.2 ms request latency. This does not establish performance
  on the phone's faster radio.
- The next phone run used HTTP on radio2, 5 GHz, channel 52, with no radio change:
  120.95 Mbps download, 90.36 Mbps upload and 9 ms latency. The busiest core
  averaged 18% busy. This is close to the user's earlier 113/88 Mbps internet
  result, but it is not the same radio as the prior 500 Mbps channel-128 test.
- A subsequent phone run on radio0, 5 GHz, channel 128, stayed on that radio
  and measured 371.52 Mbps download, 390.53 Mbps upload and 8.3 ms latency over
  HTTP. The busiest core averaged 41% busy (35% soft interrupts, 4% system,
  1% user); the other core recorded 10% system and 13% user time. The earlier
  HTTPS run on that radio measured 52.54/50.26 Mbps with a 92%-busy core.
  Throughput improved about 7.1x down / 7.8x up, with considerably more CPU
  headroom. Download remains below the user's approximate 500 Mbps internet
  result; these observations do not establish the cause of that remaining gap.
- The signed APK's 19 payload files were compared against intended source and
  generated assets. Network, firewall and uhttpd configuration hashes matched
  before and after installing the direct transfer service. No global TLS or
  hardware-crypto configuration was changed.

## 0.1.0-r6 - latest phone test diagnostics

- Completed tests report measured byte counts and durations, plus HTTP latency,
  with their authenticated finish request. The router calculates Mbps and stores
  one latest result with starting/ending radio, CPU busy percentage, and a
  per-core breakdown of user, system, interrupt and soft-interrupt CPU time.
- The record is RAM-only, replaced by the next completed test, and contains no
  session token, client address or MAC address. Interrupted runs do not overwrite
  it. Validation rejects malformed, oversized or out-of-range measurements.
- Unit and browser tests verify result calculation, replacement, cancellation,
  input rejection and agreement with the displayed browser result.
- The phone subsequently reported about 50 Mbps locally versus 500 Mbps on the
  internet on the same Wi-Fi, channel 128. The development computer has a 72 Mbps
  link on radio1, so its earlier comparison cannot reproduce that faster-radio
  discrepancy. The phone result and radio/CPU diagnostics are needed next.

## 0.1.0-r5 - radio identification and throughput investigation

- Added an authenticated, read-only connection lookup that does not start a run
  or acquire its transfer lock. Actual OpenWrt wireless status maps the client's
  associated interface to a radio name; two radios on the same band stay distinct.
- The page looks up the radio on opening, refreshes every five seconds while
  visible and idle, and refreshes on return to the tab. Unknown/failed lookups
  are explicit instead of retaining a stale radio. No traffic test starts itself.
- Unit and browser checks cover authentication, lookup without a run, lookup
  during a busy run, interface-to-radio mapping and displayed radio changes.
- A same-client comparison on radio1 (2.4 GHz, channel 13) measured roughly
  44 Mbps download / 26 Mbps upload through CGI, versus 43 / 18 through a
  temporary direct Go HTTP server. Maximum per-core average busy time was 10%
  and 8%, respectively. This did not show a throughput improvement from bypassing
  the web server. HTTP response latency did improve in that diagnostic.
- A router loopback check through the installed CGI endpoint measured about
  512 Mbps download / 1,536 Mbps upload. It excludes Wi-Fi and includes both
  server and curl work on the router, so it is only a check for a fixed server
  throughput ceiling, not a radio performance result.
- These measurements do not establish why a separate phone measured local
  50/48 Mbps versus internet 113/88 Mbps. A comparison on the same identified
  radio is still needed. No arbitrary multiplier or unproven transfer-engine
  replacement was applied. Temporary diagnostic services are not packaged.

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
