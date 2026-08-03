# Review: PR [#6030](https://github.com/gnolang/gno/pull/6030)
Event: REQUEST_CHANGES

## Body
Reviewed the six files this branch adds on top of [#6018](https://github.com/gnolang/gno/pull/6018), `git diff 5ceafd2c5..098a7b782`. Verified on 098a7b782: pointing the command at a directory with an empty `db/` returns exit 0 and leaves a `gnolang.db` behind, and flipping a byte inside a persisted `'F'` record is classified `corrupt` and exits 1.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6030-fastindex-consistency-audit/1-098a7b782/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gno.land/cmd/gnoland/fastindex_verify.go:100-103 [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L100-L103)
A data directory holding a `db/` folder but no gnoland store exits 0 here. [`LoadReadonly`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/mutable_tree.go#L519-L528) returns `(0, nil)` with no latest version, so the report reads `version=0 entries=0` and the stamp-absent branch calls that nothing to verify. For the CI gate the command is offered as, a wrong `-data-dir` or a `-db-backend` that does not match what the node ran with then stays green forever. Fail when no committed version was loaded.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6030 -R gnolang/gno

cat > gno.land/cmd/gnoland/audit_probe_test.go <<'EOF'
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/config"
	"github.com/gnolang/gno/tm2/pkg/commands"
	_ "github.com/gnolang/gno/tm2/pkg/db/goleveldb"
)

func TestAuditProbe(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, config.DefaultDBDir)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	io := commands.NewTestIO()
	io.SetOut(commands.WriteNopCloser(&out))

	err := execFastindexVerify(&fastindexVerifyCfg{dataDir: dir, dbBackend: "goleveldb"}, io)

	t.Logf("exit error: %v", err)
	t.Logf("output: %s", out.String())
	entries, _ := os.ReadDir(dbDir)
	for _, e := range entries {
		t.Logf("left behind in db/: %s", e.Name())
	}
}
EOF

go test -v -run TestAuditProbe ./gno.land/cmd/gnoland/
rm gno.land/cmd/gnoland/audit_probe_test.go
```

```
=== RUN   TestAuditProbe
    audit_probe_test.go:27: exit error: <nil>
    audit_probe_test.go:28: output: fast-index audit: version=0 stamp=0 (present=false) entries=0 mismatches=0
        no fast index present (feature disabled or never built) — nothing to verify
    audit_probe_test.go:31: left behind in db/: gnolang.db
--- PASS: TestAuditProbe (0.15s)
```
</details>

## gno.land/cmd/gnoland/fastindex_verify.go:101 [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L101)
An index whose entries are still there but whose stamp is gone prints its mismatches and then says no fast index is present, because this branch is evaluated before the mismatch branch. That state is what [line 107](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L105-L107) tells the operator to create, and what an interrupted [`dropFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/fast_index.go#L209-L215) leaves behind, since it deletes the stamp first. Exit 0 is right there; report it the way the behind case is reported.

## gno.land/cmd/gnoland/fastindex_verify.go:69 [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L69)
The database is opened read-write and created when absent, so the run leaves a `gnolang.db` in the target directory, as the repro above shows. The [help text](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L36) says READ-ONLY and the PR body offers the command for a captured node state, which is where an operator has the strongest reason to expect the directory to come back untouched. `tm2/pkg/db` has no read-only open, so say that the audit performs no writes while the database is still opened read-write.

## gno.land/cmd/gnoland/fastindex_verify_test.go:99-104 [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify_test.go#L99-L104)
Missing test: a stamp-current index that disagrees with the tree, the verdict the command exists to produce. The stamp-ahead branch is uncovered too.

<details><summary>test cases</summary>

Damaging the record needs no bptree internals, and it is also the only path that produces the [`corrupt`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/verify.go#L26) kind. This passes at 098a7b782:

```go
func TestFastindexVerify_CorruptRecord(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, config.DefaultDBDir)

	raw, err := dbm.NewDB("gnolang", testBackend, dbDir)
	require.NoError(t, err)
	prefixed := dbm.NewPrefixDB(raw, []byte(mainStorePrefix))
	tree := bptree.NewMutableTreeWithDB(prefixed, 1000, bptree.NewNopLogger(), bptree.FastIndexOption(true))

	_, err = tree.Set([]byte("k"), []byte("authoritative"))
	require.NoError(t, err)
	_, _, err = tree.SaveVersion()
	require.NoError(t, err)

	// Damage the persisted record in place, leaving the stamp current.
	fastKey := append([]byte{bptree.PrefixFast}, 'k')
	rec, err := prefixed.Get(fastKey)
	require.NoError(t, err)
	rec[len(rec)/2] ^= 0xFF
	require.NoError(t, prefixed.Set(fastKey, rec))
	require.NoError(t, raw.Close())

	out, err := runVerify(t, dir)
	assert.Error(t, err)
	assert.Contains(t, out, "corrupt")
}
```

```
=== RUN   TestFastindexVerify_CorruptRecord
    fastindex_verify_gaps_test.go:84: output: "fast-index audit: version=1 stamp=1 (present=true) entries=1 mismatches=1\n  corrupt    key=6B\n"
--- PASS: TestFastindexVerify_CorruptRecord (0.06s)
```
</details>

## tm2/pkg/bptree/verify.go:11 [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/verify.go#L11)
Missing test: more mismatches than the sample cap. No test drives past one, so nothing pins that `MismatchCount` keeps the true total while `Mismatches` stops at 256, which is what the CLI's ["and N more"](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L96-L98) line reports.

## tm2/pkg/bptree/verify.go:119 [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/verify.go#L119)
Suggestion: the audit restarts from the root for every `'F'` entry, while both the index scan and the tree are sorted by user key. [`rebuildFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/fast_index.go#L285-L301) already walks the tree in order with `iterateNodeResolved`, and the same shape here turns a random descent per entry into one pass over a full mainnet store. It also makes a key the tree holds with no index entry visible, which the current shape cannot see.

## gno.land/cmd/gnoland/fastindex_verify.go:18 [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L18)
Nit: this restates the rule inside [`constructStore`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/store/rootmulti/store.go#L593-L599), which also branches on whether the store was mounted with its own database handle. A drift between the two is silent, since an empty range reads as no index present.

## gno.land/cmd/gnoland/fastindex_verify.go:76-79 [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L76-L79)
Nit: the node cache size of 10000 arrives with no reasoning, while the package's own test helper uses 1000.
