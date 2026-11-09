#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temp=$(mktemp -d)
trap 'rm -rf "$temp"' EXIT
mkdir -p "$temp/bin" "$temp/sys/tun0"
: > "$temp/sys/tun0/tun_flags"
cat > "$temp/functions" <<'FIXTURE'
config_load() { cfg="$1"; }
config_get() {
 value="${4:-}"
 case "$cfg/$2/$3" in
 network-speed-test/main/auto_vpn) value=1; [ "$MODE" != off ] || value=0;;
 network-speed-test/main/vpn_network) [ "$MODE" != explicit ] || value='custom wan alias uplink';;
 network-speed-test/main/exclude_network) [ "$MODE" != exclude ] || value='vpn tap';;
 firewall/defaults/input) value=REJECT;;
 firewall/local/name) value=lan;;
 firewall/local/network) value=lan;;
 firewall/local/input|firewall/trusted/input) value=ACCEPT;;
 firewall/trusted/name) value=vpn;;
 firewall/trusted/network) value='vpn tap alias down ordinary';;
 firewall/outside/name) value=wan;;
 firewall/outside/network) value='wan uplink';;
 firewall/outside/input) value=ACCEPT;;
 network/lan/proto|network/ordinary/proto) value=static;;
 network/vpn/proto|network/untrusted/proto|network/alias/proto|network/down/proto|network/uplink/proto) value=wireguard;;
 network/tap/proto|network/custom/proto) value=none;;
 esac
 export "$1=$value"
}
config_get_bool() { config_get "$@"; }
config_foreach() {
 case "$cfg/$2" in
 firewall/defaults) "$1" defaults;;
 firewall/zone) for section in local trusted outside; do "$1" "$section"; done;;
 network/interface) for section in lan vpn tap untrusted wan uplink alias down ordinary custom; do "$1" "$section"; done;;
 esac
}
FIXTURE
cat > "$temp/bin/ubus" <<'FIXTURE'
#!/bin/sh
name=${2#network.interface.}
printf '%s\n' "$name"
FIXTURE
cat > "$temp/bin/jsonfilter" <<'FIXTURE'
#!/bin/sh
name=$2
if [ "$4" = '@.up' ]; then
 [ "$name" != down ] && echo true || echo false
else
 case "$name" in
 lan) echo br-lan;; vpn) echo wg0;; tap) echo tun0;; untrusted) echo wg1;;
 wan|alias) echo eth0;; uplink) echo eth1;; down) echo wg2;;
 ordinary) echo eth2;; custom) echo special0;; loopback) echo lo;;
 esac
fi
FIXTURE
chmod +x "$temp/bin/ubus" "$temp/bin/jsonfilter"
sed -e "s|^\. /lib/functions.sh$|. $temp/functions|" -e "s|/sys/class/net/|$temp/sys/|g" "$root/files/network-speed-test-networks" > "$temp/discover"
export PATH="$temp/bin:$PATH"
check() {
 actual=$(MODE="$1" sh "$temp/discover")
 [ "$actual" = "$2" ] || { printf 'FAIL %s\n%s\n' "$1" "$actual"; exit 1; }
}
check auto 'lan br-lan lan
vpn wg0 wireguard
tap tun0 vpn'
check off 'lan br-lan lan'
check exclude 'lan br-lan lan'
check explicit 'lan br-lan lan
vpn wg0 wireguard
tap tun0 vpn
custom special0 vpn'
echo 'PASS: automatic trusted VPNs, TUN detection, exclusions, manual override, disabled auto, down/untrusted/non-VPN rejection, WAN-zone and WAN-device rejection'
