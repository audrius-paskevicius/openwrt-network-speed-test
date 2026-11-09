#!/bin/sh
set -eu
BASE=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$BASE"
: "${SIGN_KEY:?Signing key outside public source required}"
APK_TOOL=${APK_TOOL:-apk}
VERSION=$(cat VERSION)
ARCH=${ARCH:-arm_cortex-a9_vfpv3-d16}
ROOT=$(mktemp -d)
trap 'rm -rf "$ROOT"' EXIT
mkdir -p "$ROOT/www/cgi-bin" "$ROOT/www/luci-static/resources/view" "$ROOT/www/luci-static/resources/network-speed-test" "$ROOT/usr/share/luci/menu.d" "$ROOT/usr/share/rpcd/acl.d" "$ROOT/usr/share/network-speed-test"
mkdir -p "$ROOT/etc/init.d"
cp dist/network-speed-test "$ROOT/www/cgi-bin/network-speed-test"
sed "s/@VERSION@/$VERSION/g" luci/network-speed-test.js > "$ROOT/www/luci-static/resources/view/network-speed-test.js"
cp luci/network-speed-test-disclaimer.js "$ROOT/www/luci-static/resources/view/network-speed-test-disclaimer.js"
sed "s/@VERSION@/$VERSION/g" luci/disclaimer.js > "$ROOT/www/luci-static/resources/network-speed-test/disclaimer.js"
cp web/disclaimer.css "$ROOT/www/luci-static/resources/network-speed-test/disclaimer.css"
cp DISCLAIMER.txt "$ROOT/www/luci-static/resources/network-speed-test/disclaimer.txt"
cp web/test.js "$ROOT/www/luci-static/resources/network-speed-test/test.js"
cp web/style.css "$ROOT/www/luci-static/resources/network-speed-test/style.css"
cp files/menu.json "$ROOT/usr/share/luci/menu.d/luci-app-network-speed-test.json"
cp files/acl.json "$ROOT/usr/share/rpcd/acl.d/luci-app-network-speed-test.json"
cp files/network-speed-test.init "$ROOT/etc/init.d/network-speed-test"
cp web/native.html web/native.js web/style.css "$ROOT/usr/share/network-speed-test/"
{ printf '%s\n' '(function(){'; sed '/^return baseclass.extend/d' web/test.js; printf '%s\n' 'window.NetworkSpeedTest={mount:mount};})();'; } > "$ROOT/usr/share/network-speed-test/engine.js"
cp LICENSE LICENSE.go-runtime README.md DISCLAIMER.txt "$ROOT/usr/share/network-speed-test/"
find "$ROOT" -type d -exec chmod 755 {} \;
find "$ROOT" -type f -exec chmod 644 {} \;
chmod 755 "$ROOT/www/cgi-bin/network-speed-test"
chmod 755 "$ROOT/etc/init.d/network-speed-test"
mkdir -p dist
"$APK_TOOL" mkpkg --files "$ROOT" --sign-key "$SIGN_KEY" --info name:luci-app-network-speed-test --info version:"$VERSION" --info arch:"$ARCH" --info license:MIT --info description:'Authenticated local browser-to-router speed test for LuCI' --info 'depends:luci-base uhttpd rpcd ubus iw' --script "post-install:$BASE/scripts/post-install" --script "post-upgrade:$BASE/scripts/post-install" --script "pre-deinstall:$BASE/scripts/pre-deinstall" --output "dist/luci-app-network-speed-test-$VERSION.apk"
