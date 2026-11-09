# Architecture

## Request flow

```mermaid
sequenceDiagram
    participant B as Browser / LuCI
    participant C as Authenticated CGI
    participant T as Interface-bound HTTP service
    B->>C: Prepare using existing LuCI session
    C->>C: Check permission; create client-bound grant
    C-->>B: Temporary test token
    B->>T: CORS preflight from original LuCI origin
    T-->>B: Allow that origin and POST headers
    B->>T: Start with token and client-generated sample
    T->>T: Recheck LuCI permission; acquire run lease
    B->>T: Pings, download and uploads over HTTP
    B->>T: Finish with measured byte counts and durations
    T-->>B: Connection and CPU diagnostics
```

The normal UI remains in LuCI. A local-network-aware browser can use a private
IP or a hostname resolving to one; requests specify `targetAddressSpace: local`.
The HTTP endpoint receives a scoped token, never the LuCI session or cookies.
Radio/path polling uses the original LuCI connection. CORS is an additional
browser restriction; every actual operation still validates authentication.

## Interfaces and VPN discovery

`files/network-speed-test-networks` selects the active `lan` interface plus
recognised VPN networks with firewall-zone input policy `ACCEPT`. It reads UCI,
netifd status and device metadata, and emits only logical name, device and kind.
It recognises WireGuard and named VPN protocols (OpenVPN, PPTP, L2TP, SSTP,
OpenConnect), plus TUN/TAP devices and WireGuard/XFRM device types. This is
detection, not a recommendation of every protocol's security.
Trust matching uses exact logical-network or device names from firewall zones;
device-name wildcards and individual per-port accept rules are not interpreted.
A zone without an explicit input policy inherits the configured default input
policy for this check (falling back to `REJECT`). Existing firewall rules still
decide whether a connection can reach a selected listener.

Defaults reject `wan`, `wan6`, `loopback`, networks in the zone named `wan`,
their currently active devices, and loopback. Devices are deduplicated. An
administrator's network/zone names and policy remain part of the trust model;
this is not a universal classifier of which connection is the internet.

Each selected device gets a procd instance with a socket bound using Linux
`SO_BINDTODEVICE`. There is no unbound fallback. Instances share the same
root-only RAM directory and file locks, so only one test runs across all paths.
Firewall rules are never changed. Interface events and network/firewall/app
configuration reloads restart discovery; a restart can interrupt a running test.

The standard configuration is automatic. For unusual VPNs with a logical netifd
interface, `/etc/config/network-speed-test` supports:

```uci
config settings 'main'
    option auto_vpn '1'
    # list vpn_network 'my_vpn'
    # list exclude_network 'unwanted_vpn'
```

`vpn_network` explicitly selects a trusted network even if automatic detection
does not recognise it; WAN exclusions still apply. `exclude_network` takes
precedence, including when excluding `lan`. Set `auto_vpn` to `0` for LAN plus
explicit VPN entries only, then run `/etc/init.d/network-speed-test restart`.
Defaults contain no private network
names. Existing firewall input restrictions still apply to explicit entries.
Device-only VPNs need a logical netifd interface; policy-based IPsec without a
tunnel interface is not automatically supported. Bridged VPN clients may be
indistinguishable from other clients on the same bridge.

Path identification follows the router's return route to the client and matches
selected VPN devices. It shows `WireGuard VPN` or `VPN connection` instead of a
local radio. Unusual asymmetric/policy routing can prevent accurate identification.
A VPN result includes the remote access network, internet route and tunnel
processing; a local-radio result measures the direct client-to-router path.

## Measurements and limits

The browser generates a 1 MiB reusable sample and a maximum 64 MiB upload buffer.
Downloads use a RAM file through Go's Linux sendfile path; uploads use splice to
discard bytes into `/dev/null`, then recycle TCP sockets. Go retains copying
fallbacks. VPN encryption is separate kernel/tunnel work and cannot be removed
by using HTTP inside the tunnel.

Latency is the median of seven HTTP exchanges after two discarded warm-ups.
Each transfer direction warms up for at least one second, then targets 6–10
seconds of measurement. Three recent throughput samples with at most 10% spread
relative to their mean allow stopping after the minimum. Upload blocks target
about 1.25 seconds and can extend the measurement target. Downloads are bounded
to 2 GiB, uploads to 64 MiB each; the browser request timeout is 18 seconds.
The server download deadline is 13 seconds and the overall run lease 90 seconds.
Hiding/leaving the page cancels; loss of connectivity may leave a lease until expiry.

CPU percentages are deltas from `/proc/stat`, averaged over the run. The UI warns
at 90% on the busiest core. They are not instantaneous CPU peaks or load averages.
When the starting interface is known, a change in any recorded connection field
(path, interface, radio, band or channel) at finish is flagged. A missing starting
interface does not establish a baseline for change detection.

## RAM state and compatibility

`/var/run/network-speed-test/` is root-only. It holds the sample, lock, current
run, scoped grant and optional latest result. Grant and run files include
sensitive session/client information. A successfully saved `last-result.json`
contains time, transport, path/radio, measurements and CPU diagnostics, without
IP/MAC/session identifiers. Result saving is best-effort. There is no accumulated
history and no upload retention. Reboot clears these RAM files.

Each new preparation replaces the idle grant, expires after 30 minutes, and is
rejected during an active run. Standalone HTTP-page Start renews its grant after
revalidating the LuCI session. The normal LuCI page obtains a fresh grant instead.
An already active native HTTP run can continue until finish/expiry after a
session is revoked; the compatibility CGI path rechecks the session on every request.

The standalone HTTP page and original authenticated CGI data operations remain
for compatibility/diagnostics. LuCI does not navigate to the standalone page.
CGI transfers require the LuCI session and, for active-run data, a scoped cookie;
they do not have the native path's low CPU cost. See [security](../SECURITY.md).
