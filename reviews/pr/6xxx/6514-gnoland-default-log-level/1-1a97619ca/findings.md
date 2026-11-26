# Findings in posting order, from round assemble: 1 to post, 0 SKIP, 3 refuted kept out

## gno.land/cmd/gnoland/README.md:64 [gh](https://github.com/gnolang/gno/blob/1a97619ca57fc8cd9a76b195f6da1e9d728e51b4/gno.land/cmd/gnoland/README.md#L64) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: the callout says older binaries default to debug without naming a release or a way to check one
Check: read README.md:62-66 for a release, version or command that shows the binary's default
Details: README.md:64-66 reads 'releases before that change defaulted to debug, so pass -log-level info explicitly on older binaries'; the README names no change, release or version for 'that change', so an operator cannot tell from it whether their binary is affected. `gnoland start -h` prints the binary's own default and settles it.
Evidence: sed -n 64,66p README.md: '> releases before that change defaulted to `debug`, so pass `-log-level info`' / '> explicitly on older binaries.'
