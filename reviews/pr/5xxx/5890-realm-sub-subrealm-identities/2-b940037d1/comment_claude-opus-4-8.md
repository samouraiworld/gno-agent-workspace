# Review: PR [#5890](https://github.com/gnolang/gno/pull/5890)
Event: REQUEST_CHANGES

## Body
The findings below land as follow-ups against master, since this merged as 5b989cad5. Each still applies at 27b5b8e24.

Two of them share a root cause worth naming: the `.gno` layer does string work the VM already exposes natively. That is where the banker gas goes, and it is why the subpath grammar now exists twice with nothing checking the two copies agree.

Reproduced on b940037d1: rewriting the sub-token gate in `NewBanker` to use `rlm.Subpath()` drops an `OriginSend` banker construction from 1,958,206 to 1,663,192 gas, against ~980k for a no-op crossing call in the same realm. Deriving a sub-address for an ephemeral `/e/…/run` host returns g1rh473gdw3gtpafql24l860tarngpkt6kjs3gks instead of panicking, so `DerivePkgSubAddr` hands back an address `Sub` will never mint. Changing the separator on both sides and regenerating goldens flips `zrealm_sub_derive_reject.gno`'s positive control from true to false with the suite still reporting ok.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5890-realm-sub-subrealm-identities/2-b940037d1/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/stdlibs/chain/banker/banker.gno:111-115 [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/banker/banker.gno#L111)
The interpreted [`chain.SplitPkgSubPath`](https://github.com/gnolang/gno/blob/b940037d1/gnovm/stdlibs/chain/address.gno#L37) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/address.gno#L37) adds ~295k gas to every `OriginSend` and `RealmIssue` banker construction: 1,958,206 vs 1,663,192 with the gate on [`rlm.Subpath()`](https://github.com/gnolang/gno/blob/b940037d1/gnovm/pkg/gnolang/uverse.go#L1776) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L1776), whose doc comment already asks consumers to prefer it over parsing `PkgPath()`. About 185k is the `strings` package load forced at `NewBanker` time and the rest is the interpreted call, against ~980k for a no-op crossing call. The two forms differ only for a pkgpath of `host#` or `#`, neither of which `Sub` can mint.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5890 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/zz_gasprobe.txtar <<'EOF'
loadpkg gno.land/r/gasprobe $WORK/gasprobe

gnoland start

gnokey maketx call -pkgpath gno.land/r/gasprobe -func Pay -send 1ugnot -gas-fee 1000000ugnot -gas-wanted 20000000 -chainid=tendermint_test test1
stdout 'OK!'
stdout 'GAS USED'

-- gasprobe/gnomod.toml --
module = "gno.land/r/gasprobe"
gno = "0.9"

-- gasprobe/gasprobe.gno --
package gasprobe

import "chain/banker"

func Pay(cur realm) {
	_ = banker.NewBanker(banker.BankerTypeOriginSend, cur)
}
EOF

echo "--- as shipped:"
go test ./gno.land/pkg/integration/ -run 'TestTestdata/zz_gasprobe' -count=1 -v 2>&1 | grep 'GAS USED'

# Answer "is this a sub-token" with the native accessor instead.
python3 - <<'PY'
p="gnovm/stdlibs/chain/banker/banker.gno"
s=open(p).read()
s=s.replace('\t\tif _, _, isSub := chain.SplitPkgSubPath(rlm.PkgPath()); isSub {',
            '\t\tif rlm.Subpath() != "" {')
open(p,"w").write(s)
PY

echo "--- with Subpath():"
go test ./gno.land/pkg/integration/ -run 'TestTestdata/zz_gasprobe' -count=1 -v 2>&1 | grep 'GAS USED'

rm gno.land/pkg/integration/testdata/zz_gasprobe.txtar
git checkout HEAD -- gnovm/stdlibs/chain/banker/banker.gno
```

```
--- as shipped:
        GAS USED:   1981360
--- with Subpath():
        GAS USED:   1658032
```
</details>

## gnovm/stdlibs/chain/address.gno:41-43 [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/address.gno#L41)
This says `DerivePkgSubAddr` can never derive an address `cur.Sub` would refuse, but it mirrors only [`subRealmPathError`](https://github.com/gnolang/gno/blob/b940037d1/gnovm/pkg/gnolang/uverse.go#L499) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L499), and `Sub` separately refuses ephemeral hosts at [`uverse.go:1751`](https://github.com/gnolang/gno/blob/b940037d1/gnovm/pkg/gnolang/uverse.go#L1751) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L1751) with no mirror, so the helper derives spendable-looking addresses for `/e/…/run` hosts that nothing can ever mint. Exposure is small, an ephemeral realm lives one transaction, but this comment is what a maintainer will trust when deciding whether the next `Sub` guard needs mirroring.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5890 -R gnolang/gno

cat > gnovm/tests/files/zz_ephem_probe.gno <<'EOF'
// PKGPATH: gno.land/r/demo/probe
package probe

import "chain"

func main(cur realm) {
	defer func() {
		if r := recover(); r != nil {
			println("REJECTED:", r)
		}
	}()
	println("derived:", chain.DerivePkgSubAddr(
		"gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run", "treasury"))
}

// Output:
// x
EOF

go test -run 'TestFiles/zz_ephem_probe.gno$' ./gnovm/pkg/gnolang/ -count=1 2>&1 | grep -E 'derived|REJECTED'
rm gnovm/tests/files/zz_ephem_probe.gno
```

```
+derived: g1rh473gdw3gtpafql24l860tarngpkt6kjs3gks
```
</details>

## gnovm/tests/files/zrealm_sub_derive_reject.gno:52-54 [↗](../../../../../.worktrees/gno-review-5890/gnovm/tests/files/zrealm_sub_derive_reject.gno#L52)
The positive control hardcodes the `#` literal while `addr` comes from `subRealmSep`, so a future separator change plus `-update-golden-tests` rewrites the golden from true to false and the suite still reports ok: the control then asserts the opposite of what its comment says it guards. This is the same shape as the hardcoded `:dao/42` in `zrealm_sub0` that 3127b465d's message records catching during regeneration.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5890 -R gnolang/gno

# Simulate any future separator change, applied to both mirrors.
sed -i 's/^const subRealmSep = "#"/const subRealmSep = "%"/' \
  gnovm/pkg/gnolang/uverse.go gnovm/stdlibs/chain/address.gno

go test -run 'TestFiles/zrealm_sub_derive_reject.gno$' ./gnovm/pkg/gnolang/ \
  -update-golden-tests -count=1 2>&1 | tail -1
grep 'run-shaped subpath hashed' gnovm/tests/files/zrealm_sub_derive_reject.gno

git checkout HEAD -- gnovm/pkg/gnolang/uverse.go gnovm/stdlibs/chain/address.gno \
  gnovm/tests/files/zrealm_sub_derive_reject.gno
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.772s
	println("run-shaped subpath hashed (not extracted):",
// run-shaped subpath hashed (not extracted): false
```
</details>

## gnovm/stdlibs/chain/banker/banker.gno:80-81 [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/banker/banker.gno#L80)
"anyone holding rlm can construct a banker for it" is the contract the [`IsCurrent` gate](https://github.com/gnolang/gno/blob/b940037d1/gnovm/stdlibs/chain/banker/banker.gno#L101) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/banker/banker.gno#L101) twenty lines below exists to break. It is the first line a realm author reads about `NewBanker`, and it now describes the hole this PR closes.

## gnovm/stdlibs/chain/address.gno:61-92 [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/address.gno#L61)
Missing test: the subpath grammar and separator exist twice, here and in [`uverse.go:519-550`](https://github.com/gnolang/gno/blob/b940037d1/gnovm/pkg/gnolang/uverse.go#L519-L550) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L519), tied only by "keep in sync" comments, and no test drives one input through both. Both copies agree at b940037d1; teaching only this one to accept non-ASCII leaves the entire `TestFiles` suite green. The shipped reject sets are two hand-maintained lists that never try `a#b`, NUL, non-ASCII, RTL-override, `.`, `a/./b`, or edge punctuation on both sides.

<details><summary>test cases</summary>

Two tests close it, both green at b940037d1. [`zz_subpath_mirror_sync_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5890-realm-sub-subrealm-identities/2-b940037d1/tests/zz_subpath_mirror_sync_test.go) · [↗](tests/zz_subpath_mirror_sync_test.go) parses both files and compares the two grammars' ASTs, failing on any lone drift. [`zrealm_sub_mirror_vector.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5890-realm-sub-subrealm-identities/2-b940037d1/tests/zrealm_sub_mirror_vector.gno) · [↗](tests/zrealm_sub_mirror_vector.gno) drives `TestIsValidSubpath`'s vector verbatim through `cur.Sub` and `chain.DerivePkgSubAddr`, asserting per-input agreement:

```
go test ./gnovm/pkg/gnolang/ -run TestSubpathGrammarMirrorInSync -v
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_sub_mirror_vector.gno$' -v
```
</details>

## gnovm/stdlibs/chain/address.gno:49 [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/address.gno#L49)
Nit: `len(pkgpath)+1+len(subpath)` hardcodes a one-byte separator where [the Go side](https://github.com/gnolang/gno/blob/b940037d1/gnovm/pkg/gnolang/uverse.go#L503) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L503) measures `len(synthesized)`, so the two caps diverge silently the day `subRealmSep` stops being one byte. `len(pkgpath)+len(subRealmSep)+len(subpath)` tracks the constant.

## gnovm/tests/files/zrealm_sub_derive_reject.gno:49 [↗](../../../../../.worktrees/gno-review-5890/gnovm/tests/files/zrealm_sub_derive_reject.gno#L49)
Nit: "the safety rests on the host-colon-free assert" describes the pre-`#` code; [the assert](https://github.com/gnolang/gno/blob/b940037d1/gnovm/stdlibs/chain/address.gno#L51) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/address.gno#L51) is now host-`#`-free.

## gnovm/tests/files/zrealm_sub_validation.gno:32 [↗](../../../../../.worktrees/gno-review-5890/gnovm/tests/files/zrealm_sub_validation.gno#L32)
Nit: `// synthesized > 256 (host+':'+256)` still names `:` as the separator.

## gnovm/stdlibs/native_gas_test.go:15 [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/native_gas_test.go#L15)
Nit: `TestSubRealmGasMirrorsPackageAddress` pins only `Base` and `Slope`, but `Sub` also pays the flat `OpCPUCallNativeBody` that `chargeNativeGas` levies on every uverse native, so its cost sits ~2205 above the `packageAddress` the name claims it mirrors. The [code comment](https://github.com/gnolang/gno/blob/b940037d1/gnovm/pkg/gnolang/uverse.go#L1726-L1735) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L1726) now says this plainly; the name is the remaining overstatement.

## docs/resources/gno-interrealm-v2.md:452 [↗](../../../../../.worktrees/gno-review-5890/docs/resources/gno-interrealm-v2.md#L452)
Nit: §5.5 shows a `RealmSend` banker built from a sub-token but never mentions that [`RealmIssue` and `OriginSend` panic](https://github.com/gnolang/gno/blob/b940037d1/gnovm/stdlibs/chain/banker/banker.gno#L111-L115) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/banker/banker.gno#L111) for one, which is the only banker rule specific to sub-realms.
</content>
