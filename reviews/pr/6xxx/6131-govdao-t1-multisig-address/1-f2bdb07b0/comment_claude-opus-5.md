# Review: [#6131](https://github.com/gnolang/gno/pull/6131)
Event: REQUEST_CHANGES

## Body
`test13` deploys `r/sys/names` and `boards2/v1` from this repository's [`examples/` tree](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L637), not from gnoland1's chain state. Eight of test13's caller fields matched those realms' admin constants at the merge base and no longer do:

- [`names-enable/meta.json`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/migration/names-enable/meta.json#L5) against [`admin`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L59): `Enable` [panics with `caller is not admin`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/sys/names/verifier.gno#L117-L119) and namespace enforcement stays off for the life of that chain.
- The seven [`boards2-cascade`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json#L20) callers against [`gPerms`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/boards.gno#L47): [`WithPermission`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/boards2/v1/public.gno#L146) rejects the transactions those patches exist to admit.

Repointing the callers is not the fix. [The script picks the multisig](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/gen-genesis.sh#L1535) because it is the only account replayed history is guaranteed to fund. The caller now has to be both funded and the realm admin.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6131 -R gnolang/gno
base=$(git merge-base origin/master HEAD)

show() { # <rev> <label>
  printf '%s\n' "$2"
  printf '  verifier.gno admin      %s\n' "$(git show "$1":examples/gno.land/r/sys/names/verifier.gno | sed -n 's/.*admin[[:space:]]*=[[:space:]]*address("\(g1[0-9a-z]*\)").*/\1/p')"
  printf '  boards2/v1 gPerms       %s\n' "$(git show "$1":examples/gno.land/r/gnoland/boards2/v1/boards.gno | sed -n 's/.*initRealmPermissions("\(g1[0-9a-z]*\)").*/\1/p')"
  printf '  test13 NAMES_ADMIN      %s\n' "$(git show "$1":misc/deployments/test13.gno.land/gen-genesis.sh | sed -n 's/^NAMES_ADMIN=\(g1[0-9a-z]*\).*/\1/p')"
  printf '  test13 names-enable     %s\n' "$(git show "$1":misc/deployments/test13.gno.land/transactions/migration/names-enable/meta.json | sed -n 's/.*"caller_override"[^"]*"\(g1[0-9a-z]*\)".*/\1/p')"
  printf '  test13 cascade h126810  %s\n' "$(git show "$1":misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json | sed -n 's/.*"caller"[^"]*"\(g1[0-9a-z]*\)".*/\1/p' | head -1)"
}
show "$base" "merge base"
show HEAD "PR head"
```

All five values agree at the merge base. At the head the last three keep the old address while the two constants they mirror move.

```
merge base
  verifier.gno admin      g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
  boards2/v1 gPerms       g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
  test13 NAMES_ADMIN      g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
  test13 names-enable     g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
  test13 cascade h126810  g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
PR head
  verifier.gno admin      g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf
  boards2/v1 gPerms       g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf
  test13 NAMES_ADMIN      g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
  test13 names-enable     g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
  test13 cascade h126810  g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
```

The `boards2-cascade` group is itself the evidence that `test13` runs current `examples/` code. [Its README row](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/test13.gno.land/README.md?plain=1#L102) attributes the group to [PR 5280](https://github.com/gnolang/gno/pull/5280) editing `initRealmPermissions` in this repository, which could not have reached a chain running immutable historical code.

A caller that misses the constant fails loudly rather than silently. Reverting the [`patchpkg` line of `sys_namereg_v1_controller.txtar`](https://github.com/gnolang/gno/blob/f2bdb07b0/gno.land/pkg/integration/testdata/sys_namereg_v1_controller.txtar#L24) to the old address, with the realm source left at this head, reddens the test:

```
--- FAIL: TestTestdata/sys_namereg_v1_controller (8.01s)
        > gnokey maketx call -pkgpath gno.land/r/sys/names -func Enable ...
        Data: caller is not admin
        panic: caller is not admin
        Enable at gno.land/r/sys/names/verifier.gno:118
```

Sweep coverage for the rest of the change: 407 occurrences of the old address at the merge base, 393 rewritten, 14 left under `misc/deployments/gnoland1/` and `misc/deployments/test13.gno.land/`. Added and removed lines are equal as multisets once both addresses map to one token, apart from three `gen-genesis.sh` comment lines dropping the word `gnoland1`. No file holds both addresses, and no hex or base64 encoding of either address's bytes appears in the tree.

</details>

## misc/deployments/topaz.gno.land/gen-genesis.sh:137 [gh](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L137) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/topaz.gno.land/gen-genesis.sh#L137)
Suggestion: the script never compares `NAMES_ADMIN` against the `caller_override` that becomes the real caller, so a half-applied swap would print [one address](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L722) and cut genesis with another. Copy the three-line comparison `pearl` and `sapphire` run at [`pearl/gen-genesis.sh:798-800`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L798-L800).
