# Network Speed Test

[![Checks](https://github.com/audrius-paskevicius/openwrt-network-speed-test/actions/workflows/check.yml/badge.svg)](https://github.com/audrius-paskevicius/openwrt-network-speed-test/actions/workflows/check.yml)
[![Release](https://img.shields.io/github/v/release/audrius-paskevicius/openwrt-network-speed-test)](https://github.com/audrius-paskevicius/openwrt-network-speed-test/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

<!-- project-disclaimer:start -->
> [!WARNING]
> EXPERIMENTAL, AI-GENERATED SOFTWARE — USE AT YOUR OWN RISK
>
> This is a free, open-source project. The person behind it supplied the ideas, requirements, and direction; AI generated the code. That person cannot independently verify every aspect of its implementation or behaviour. Testing does not guarantee correctness, reliability, or safety.
>
> The software generates network traffic between your device and the router. Tests may temporarily consume available network capacity and processing resources, interrupt other traffic, or produce misleading measurements. Errors may cause network disruption, security weaknesses, data loss, or other harm.
>
> It is provided “as is,” without promised support, maintenance, or repair. To the fullest extent permitted by applicable law, the provider excludes warranties and liability arising from its use.
>
> Use only if you understand and accept the risks, have suitable backups and recovery arrangements, and can bear the resulting disruption, costs, or losses. Do not rely on it where failure could cause harm.
<!-- project-disclaimer:end -->

A small OpenWrt APK providing **Network → Speed Test** in LuCI. Sign in to the
router, open the page on the device you want to measure, and press **Start**.
Download, upload and local request latency are measured between that browser and
the router, directly or through a supported VPN terminating there. Nothing
contacts an internet speed-test service. There is one button;
warm-up and measurement lengths are automatic.

The connected OpenWrt radio name, band and channel appear when the page opens and
refresh every five seconds while it is visible and idle. Radio names come from
OpenWrt wireless status, not guesses based on interface numbers. If the router
cannot identify a direct Wi-Fi association, the page says so.
For a detected VPN path, it shows the VPN connection instead of a radio name.
That result includes the remote access connection, route and VPN processing;
it does not isolate the router's Wi-Fi radio.

The page uses native LuCI tables, buttons, progress bars and messages, following
the active router theme. The LuCI page stays open over HTTPS while the bulk test
traffic goes directly over local HTTP, avoiding TLS work on the router.

Results are practical
measurements of the complete client-to-router path, not a guarantee of maximum
radio capacity. Browser, client, router CPU, signal and competing traffic can
limit throughput. A wired client measures its Ethernet path. Latency is HTTP
request/response latency, including router processing, rather than ICMP ping.

## Install and use

The published APK targets OpenWrt 25.12 with LuCI and
`arm_cortex-a9_vfpv3-d16` (developed on a Linksys WRT3200ACM). Other CPU
architectures need their own build. This is an APK, not an older OpenWrt IPK.

1. Download the APK, `network-speed-test-public.pem` and `SHA256SUMS` from the
   [latest release](https://github.com/audrius-paskevicius/openwrt-network-speed-test/releases/latest).
   Compare the downloaded files' SHA-256 hashes with `SHA256SUMS`. Review the
   [disclaimer](DISCLAIMER.txt) and [security model](SECURITY.md).
2. Copy the public key to `/etc/apk/keys/` on the router and the matching APK
   to `/tmp/`. Installing a signing key trusts packages signed by that key.
3. On the router, verify and install the downloaded package:

   ```sh
   apk verify /tmp/luci-app-network-speed-test-0.2.2-r1.apk
   apk add /tmp/luci-app-network-speed-test-0.2.2-r1.apk
   ```

4. Refresh LuCI, sign in normally, and open **Network → Speed Test** on the
   device to measure. Acknowledge the notice and press **Start**. Keep that
   page visible until it finishes. If Chrome asks for local network access,
   allow it for the router page. No second router login is required.

Trusted VPN interfaces are discovered automatically; no settings tab is needed.
Connect the VPN, then open LuCI using the router's private address or a name
resolving to it (including a local single-label hostname). The service follows
existing firewall input permissions. See [VPN discovery and overrides](docs/ARCHITECTURE.md#interfaces-and-vpn-discovery)
for supported interface types and uncommon configurations.

An upgrade uses the same `apk add` command with the newer package. To remove:

```sh
apk del luci-app-network-speed-test
```

Removal stops and disables the test service. Modified UCI configuration may be
retained by APK's configuration-file handling. If no longer needed, remove its
public signing key separately from `/etc/apk/keys/`. The latest diagnostic result
and expired run/lock files can remain in RAM until reboot; browser acknowledgement
is site data and is not removed by APK uninstall.

## Disclaimer

A permanent tab shows the full project-specific notice. The first visit requires
acknowledgement, remembered for the exact text in this browser while site data is
retained. New logins, browser/computer restarts and upgrades do not ask again when
the text is unchanged. Clearing site data or changing the notice asks again. If
storage is blocked, acknowledgement lasts for the active page module only. This
is a UI acknowledgement; server-side LuCI session authorization always applies.

## Operation

- LuCI handles the normal router login and disclaimer. Start keeps the current
  LuCI page open, including HTTPS, while transferring test data directly over
  local HTTP using a random, client-bound capability. The router password
  and LuCI session token are never passed to the HTTP test service. Its token can only run
  speed tests, expires after 30 minutes, and cannot mint other tokens or access
  router administration. Starting a new run rechecks the original LuCI session.
- HTTPS-to-HTTP requests require a browser with the Local Network Access
  mixed-content exception (tested in desktop Chrome 157). Allow local network
  access if prompted. Unsupported browsers show an error without redirecting
  or silently changing the measurement to HTTPS. CORS permits only the exact
  LuCI origin that issued the current client-bound grant; cookies are omitted
  from transfer requests. Idle radio lookup stays on the LuCI connection.
- The browser supplies a 1 MiB data sample, generates upload buffers, measures
  bytes and elapsed time, chooses test lengths, calculates throughput and
  latency, and renders results. Reloading the test page does not run a test.
- A native HTTP service listens on port 18080, with one instance bound to each
  selected LAN/VPN device using Linux SO_BINDTODEVICE. No firewall rule or global web-server setting is
  changed. It serves the test and public LuCI styling assets; all measurement
  operations require the scoped token and matching client address.
- The data sample lives in OpenWrt RAM storage, not flash. Downloads reuse this
  file via Linux sendfile. Uploads drain the small initial HTTP buffer, then use
  Go's Linux splice path from the TCP socket to /dev/null. The socket then
  returns to the HTTP server so consecutive uploads retain their TCP connection. There is no payload
  encryption, CGI pipe relay, per-packet generation or retained upload data in
  the normal test path. Go has safe copy fallbacks where a kernel path is absent.
- Each direction warms up for at least a second and targets six to ten seconds
  of measurement (an in-flight upload block may extend that target),
  and is bounded by transfer timeouts and one exclusive run lease. Leaving or
  hiding the page interrupts the test. Payload work is bounded to 2 GiB per
  download and 64 MiB per upload request.
- The server attempts to save the latest completed browser result, radio
  before/after, transport and CPU diagnostics in RAM for troubleshooting. A
  successfully saved test replaces it; reboot clears it. Saving can fail without
  invalidating the result displayed in the browser. Results contain no IP, MAC
  or session token. The private
  RAM access grant necessarily holds the original session for revalidation and
  is readable only by root. No accumulated result history is kept.
- Radio identity is checked at the start and finish; a change is reported. A busy
  CPU core is reported as a possible limit. This is still a client-to-router test,
  so CPU/network driver work cannot be eliminated altogether. It is not a test
  of internet service or a guarantee of maximum radio capacity.
- The original authenticated CGI transfer operations remain for diagnosis and
  compatibility; the normal UI uses the direct HTTP path. Uninstall stops the
  native service. The permanent disclaimer tab remains in LuCI.

Only one test can run on the router at a time. Preparing a test replaces any
previous idle access grant; an active run prevents a new preparation. The normal
LuCI page obtains a fresh grant on every Start. The older standalone HTTP page
remains for compatibility and is no longer opened by LuCI.

## Troubleshooting

- **Local connection blocked:** use a browser supporting the local-network
  mixed-content exception; desktop Chrome 157 was tested, and the user also
  reported success with the updated UI. Check the router page's local-network permission
  and that the test service is running on the selected LAN/VPN path. Do not disable browser security or
  expose port 18080 to the internet to work around this.
- **Another test is running:** let that test finish. An abandoned run expires
  after 90 seconds; returning to the page and pressing Start obtains fresh access.
- **Radio unavailable:** wired clients, routed paths, mesh/extender clients or
  unavailable neighbour/hostapd data may not identify a direct radio association.
- **Unexpected throughput:** check the displayed radio/channel and repeat under
  similar conditions. Values measure the client, connection and router together.
  The busy warning means the busiest CPU core averaged at least 90% during the
  run; it is not a load-average reading or proof of the cause of a low result.

For local diagnostics, root can read
`/var/run/network-speed-test/last-result.json`. Avoid sharing raw router state:
`grant.json` contains a LuCI session and `run.json` includes a client address.
See [architecture](docs/ARCHITECTURE.md) and [verification evidence](docs/TESTING.md).

## Build and verification

Current target: OpenWrt 25.12, ARMv7 mvebu/cortexa9 (`arm_cortex-a9_vfpv3-d16`).
Use Go 1.26 or later, Node.js 24 or later for browser tests, and native APK tools
v3 with `mkpkg`. Building a different architecture requires matching both the Go
target and APK architecture metadata.

```sh
go test ./...
go vet ./...
node tests/check-js.cjs
node tests/disclaimer.cjs
node tests/check-docs.cjs
sh tests/network-discovery.sh
mkdir -p dist
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/network-speed-test ./cmd/network-speed-test
SIGN_KEY=/private/signing.pem APK_TOOL=/path/to/apk.static sh scripts/build-apk.sh
apk --keys-dir "$(pwd)/release" verify dist/luci-app-network-speed-test-$(cat VERSION).apk
```

`node tests/browser-ui.cjs` uses a disposable Chrome with remote debugging on
localhost:9223 (`CHROME_PORT` overrides it). Its traffic is synthetic. Backend
tests cover authentication, request methods/origin, byte bounds, incomplete
uploads, expiry, exclusivity, disconnects and connection-change reporting.

The repository public key verifies the maintainer's releases. For your own
build, generate a separate signing key and use its matching public key when
verifying/installing; see [contributing and release steps](CONTRIBUTING.md).
No Wi-Fi, network or firewall configuration is changed by package installation.

Only tracked sources belong in a public source archive. `.private/`, `.tools/`,
`dist/`, signing secrets, deployment logins and browser profiles are excluded.

See the [changelog](CHANGELOG.md) for releases. Report reproducible bugs through
[GitHub Issues](https://github.com/audrius-paskevicius/openwrt-network-speed-test/issues);
follow [SECURITY.md](SECURITY.md) for vulnerabilities.
