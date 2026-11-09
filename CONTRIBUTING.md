# Contributing and releases

Keep the UI small, use LuCI controls, and preserve the single Start action and
normal router login. Avoid moving payload work back into CGI/TLS, retaining
uploaded data or changing router firewall/network settings automatically.

## Checks

Use Go 1.26 or newer and Node.js 24 or newer. Run:

```sh
go test ./...
go vet ./...
node tests/check-js.cjs
node tests/disclaimer.cjs
node tests/check-docs.cjs
sh tests/network-discovery.sh
```

On a supported Go race-detector platform also run `go test -race ./...`.
For the browser fixture, start disposable Chrome with remote debugging bound to
localhost on port 9223, then run `node tests/browser-ui.cjs`. Set `CHROME_PORT`
for a different port. Do not expose Chrome's debugger to the network or reuse
your personal browser profile. The fixture does not measure network performance.

Follow the [README build commands](README.md#build-and-verification). Go and APK
architecture targets must agree. The official release targets ARMv7 Cortex-A9;
building another target does not establish that it works on that router.

Generate your own signing key outside the source tree for custom packages:

```sh
openssl genrsa -out /private/my-speed-test.pem 4096
openssl rsa -in /private/my-speed-test.pem -pubout -out /private/my-speed-test-public.pem
SIGN_KEY=/private/my-speed-test.pem APK_TOOL=/path/to/apk.static sh scripts/build-apk.sh
```

Verify custom packages with the matching custom public key directory. The
repository public key is for official release verification, not custom builds.

## Release procedure

1. Update `VERSION`, installation examples, changelog and verification evidence.
   Keep `DISCLAIMER.txt` and the README notice exactly aligned. Record meaningful
   limitations and distinguish current behaviour from historical results.
2. Complete checks, build for the target, and run the backend tests on ARM.
   Exercise the real LuCI/HTTP path and relevant VPN paths. Never substitute
   fixture throughput for a real measurement.
3. Build and verify the signed APK. Extract it and compare every payload file to
   intended sources/generated output; inspect scripts and permissions. Include
   user documentation, licences and the canonical disclaimer.
4. Scan **all reachable Git history**, the source archive, extracted APK and
   release notes for secrets/private data. Use a secret scanner plus review;
   neither proves absence of every secret. Never upload `.private/`, `.tools/`,
   signing private keys, browser/session data, router backups or capture files.
5. Commit the final sources. Tag the exact commit and create its source ZIP with
   `git archive`; do not ZIP the workspace. Create SHA-256 checksums for the APK,
   source ZIP, public signing key and disclaimer. Verify archive contents again.
6. Publish a GitHub release with those five assets, clear target/installation
   notes, changes, verification evidence and limitations. Confirm checks on the
   published commit, and download assets again to verify their hashes.

Report ordinary bugs through Issues using sanitised examples. Follow
[SECURITY.md](SECURITY.md) for vulnerabilities. Contributions do not create a
support or maintenance obligation.
