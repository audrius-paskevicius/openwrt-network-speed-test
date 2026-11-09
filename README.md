# Network Speed Test

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
the router. Nothing contacts an internet speed-test service. There is one button;
warm-up and measurement lengths are automatic.

The connected OpenWrt radio name, band and channel appear when the page opens and
refresh every five seconds while it is visible and idle. Radio names come from
OpenWrt wireless status, not guesses based on interface numbers. If the router
cannot identify a direct Wi-Fi association, the page says so.

The page uses native LuCI tables, buttons, progress bars and messages, following
the active router theme.

Results are practical
measurements of the complete client-to-router path, not a guarantee of maximum
radio capacity. Browser, client, router CPU, signal and competing traffic can
limit throughput. A wired client measures its Ethernet path. Latency is HTTP
request/response latency, including router processing, rather than ICMP ping.

## Disclaimer

A permanent tab shows the full project-specific notice. The first visit requires
acknowledgement, remembered for the exact text in this browser while site data is
retained. New logins, browser/computer restarts and upgrades do not ask again when
the text is unchanged. Clearing site data or changing the notice asks again. If
storage is blocked, acknowledgement lasts for the active page module only. This
is a UI acknowledgement; server-side LuCI session authorization always applies.

## Operation

- Each direction warms up for at least one second. Measurements normally last
  six seconds, extending to ten when recent readings vary. Slow individual
  requests can extend elapsed time; each CGI process has a 20-second hard limit.
- Download uses an uncompressed, reused random buffer. Uploads are adaptively
  sized up to 32 MiB and are counted only after the router confirms receipt.
- Payloads are streamed/discarded in memory. A tiny exclusive-run lease lives in
  `/var/run/network-speed-test` (OpenWrt RAM storage). No payload or results are
  written to flash. The latest completed result (client-reported byte counts,
  timings, radio before/after and router CPU usage) is kept in RAM for diagnosis.
  A new completed result replaces it; reboot clears it. It contains no client IP,
  MAC address or session token. There is no accumulated result history.
- The existing LuCI web server serves the CGI endpoint. No extra listener,
  firewall opening, scheduled test or background daemon is installed.
- Every request requires a live LuCI session with the package ACL. The session
  token is sent in the Authorization header, never in a URL. A short-lived,
  HTTP-only run cookie is bound to the session and requesting IP address.
- Only one run is accepted at a time. Leaving or hiding the page interrupts the
  measurement; incomplete results are cleared. Abandoned leases expire in 90 seconds.
- If the requesting Wi-Fi neighbour can be identified, the band/channel is shown.
  A different association at the end is reported. This comparison does not detect
  every brief roam away and back. A busy router CPU is reported as a possible limit.

The transfer engine is a small Go CGI executable using only the standard library.
It reuses OpenWrt's web server and session authorization. Each transfer request
starts a CGI process; adaptive upload blocks amortize that overhead. Validate on
the actual hardware before treating results as a radio-capacity benchmark.

## Build and verification

Current target: OpenWrt 25.12, ARMv7 mvebu/cortexa9 (`arm_cortex-a9_vfpv3-d16`).
Use Go 1.26 or later, Node.js 24 or later for browser tests, and native APK tools
v3 with `mkpkg`. Building a different architecture requires matching both the Go
target and APK architecture metadata.

```sh
go test ./...
node tests/check-js.cjs
node tests/disclaimer.cjs
mkdir -p dist
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/network-speed-test ./cmd/network-speed-test
SIGN_KEY=/private/signing.pem APK_TOOL=/path/to/apk.static sh scripts/build-apk.sh
apk --keys-dir "$(pwd)/release" verify dist/luci-app-network-speed-test-$(cat VERSION).apk
```

`node tests/browser-ui.cjs` uses a disposable Chrome with remote debugging on
localhost:9223 (`CHROME_PORT` overrides it). Its traffic is synthetic. Backend
tests cover authentication, request methods/origin, byte bounds, incomplete
uploads, expiry, exclusivity, disconnects and connection-change reporting.

Install the matching public signing key in `/etc/apk/keys/`, then install the
signed APK using LuCI Software or `apk add ./luci-app-network-speed-test-VERSION.apk`.
Refresh LuCI after installation. No Wi-Fi, network or firewall settings are changed.

Only tracked sources belong in a public source archive. `.private/`, `.tools/`,
`dist/`, signing secrets, deployment logins and browser profiles are excluded.
