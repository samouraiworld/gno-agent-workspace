#!/bin/sh
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
# judge-173-out-scratch.sh
#
# Asserts what misc/gnopreview writes into -out, the directory pr-preview.yml
# uploads as the published site, and what it hands gnodev as -home.
# Measured at head ecf7af0f29abe4737a52803d672bc5a33c17cc60: -out holds
# gnodev.log, and -home is passed as the RELATIVE path "_preview/.gnodev-8899"
# alongside "-C <root>/examples"; no .gnodev-* directory is ever created.
#
# From a plain clone of gnolang/gno:
#   gh pr checkout 6194 -R gnolang/gno && git checkout ecf7af0f2
#   sh judge-173-out-scratch.sh
set -eu
root=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp" "$root/_preview"' EXIT

(cd "$root/misc/gnopreview" && go build -o "$tmp/gnopreview" .)

# A stub gnodev: record the argv gnopreview builds, then exit, so the run costs
# no node boot. waitReady sees the process die and render() returns.
cat > "$tmp/stub-gnodev" <<EOF
#!/bin/sh
echo "CWD=\$(pwd)" > "$tmp/argv.txt"
for a in "\$@"; do echo "ARG=\$a" >> "$tmp/argv.txt"; done
exit 0
EOF
chmod +x "$tmp/stub-gnodev"

# A gnoweb-only diff: the plan seeds four realms, so render() reaches startGnodev.
echo 'gno.land/pkg/gnoweb/app.go' > "$tmp/changed.txt"
(cd "$root" && "$tmp/gnopreview" render -root "$root" -changed "$tmp/changed.txt" \
    -out _preview -gnodev "$tmp/stub-gnodev" -timeout 5s || true)

echo "--- -home as passed to gnodev (relative; -C chdirs first) ---"
grep -A1 -- '-home' "$tmp/argv.txt"
grep -A1 -- '^ARG=-C$' "$tmp/argv.txt"
echo "--- what landed in the published directory ---"
ls -a "$root/_preview"
echo "--- any .gnodev-* directory, anywhere under the tree ---"
find "$root/_preview" "$root/examples" -maxdepth 2 -name '.gnodev-*' -print
echo "(no line above: gnodev's -home is import-only — setup_address_book.go:18"
echo " warns and skips when the directory does not exist, and creates nothing)"
