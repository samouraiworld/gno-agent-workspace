# PR #5438: feat(tm2): immutable B+32 tree — drop-in replacement for IAVL

URL: https://github.com/gnolang/gno/pull/5438
Author: jaekwon | Base: master | Files: 100 | +27894 -86
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: `51892161f` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5438 51892161f`

Round 2. Round 1 reviewed head `33f5dc340`; this round reviews the squash-merged code now in master (`51892161f`). The round-1 Criticals are resolved: PR-scope / red-CI are moot post-merge, and the data-loss/panic bugs (`VersionExists` error-swallow, idempotent-save leaking into `Rollback`, `deleteAllNodesForVersion`, `getChild` panic-on-not-found, `Iterate` swallowing resolver errors) are all fixed in the merged code (verified below).

**TL;DR:** This adds a from-scratch versioned Merkle tree (a B+ tree with 32-way nodes) as an alternative to the existing IAVL tree for storing blockchain state, with shorter proofs and far fewer disk reads/writes per operation. It is merged but NOT yet used by the chain (production still mounts IAVL); its proof decoder is registered, and its state-sync importer exists but is not wired to any peer-facing handler.

**Verdict: MERGED — no exploitable defect found in a deep adversarial pass.** Five lenses (proof-soundness, determinism, pruning/state-corruption, memory-safety/DoS, coverage) plus targeted fuzzing and multi-thousand-op differential/stress testing found no proof forgery, no honest-node consensus fork, no pruning data-loss, and no reachable memory-safety bug. The findings are four latent hardening items for a follow-up PR, all gated on bptree actually being mounted (or state-sync wired): a fragile compressed-proof crash barrier, an unbounded importer, separators living outside the Merkle commitment, and an empty-key behavior divergence from IAVL.

## Summary

The bptree is a deterministic, copy-on-write, versioned Merkle tree. Each node carries an in-memory binary "mini-merkle" of depth 5 over its 32 slots; leaf-slot hashes are `SHA256(0x00‖…)`, inner hashes `SHA256(0x01‖…)`, and empty slots a sentinel `SHA256(0x02)` that short-circuits so an empty subtree at any depth collapses to the same value. That uniform structure maps cleanly onto an ICS23 binary-merkle ProofSpec with `EmptyChild = sentinel`, giving membership and non-membership proofs. Values are stored out-of-line by content hash; nodes and values are addressed by `{version, nonce}` keys, never content, so copy-on-write shares unchanged records by reference and pruning is a dual-tree-walk that deletes records of version `v` not referenced by its successor. An optional, off-by-default fast index trades a duplicated value for a one-read committed `Get`, explicitly outside the Merkle commitment.

The adversarial result is the headline: the soundness-critical machinery holds. Domain separation defeats leaf/inner confusion; the sentinel mapping makes "empty" provable and unforgeable; the ICS23 spec bounds depth (`MinDepth 5`, `MaxDepth 60`) and the outer `merkle.ProofRuntime` checks the proof-computed root against the trusted app hash, so a self-computed root in `CommitmentOp.Run` is not a forgery hole. The order-dependent B+ shape is consensus-safe because the cache store sorts each block's write set before it reaches the tree. Everything below is hardening, not a live exploit.

## Glossary

- **mini-merkle**: in-memory depth-5 binary merkle over a node's 32 slots; its root is the node's `Hash()`. Not serialized; rebuilt on load.
- **sentinel**: `SHA256(0x02)`, the hash of an empty slot/subtree; ICS23 `EmptyChild`. `HashInner(sentinel, sentinel)=sentinel` short-circuits so empty subtrees collide at every depth.
- **NodeKey / ValueKey**: `{version:8, nonce:4}` identifiers for a stored node (`B…`) / out-of-line value (`V…`). Not content hashes, so copy-on-write shares by record identity.
- **dual-tree-walk pruning**: deleting version `v` by descending `v`'s and its successor's roots together and deleting only records (by NodeKey) the successor no longer references.
- **fast index**: optional `F‖userKey → version‖value` accelerator outside the Merkle commitment; a hit is trusted only when its version ≤ the reader's snapshot.
- See `docs/glossary.md` for app hash, copy-on-write, IAVL, ICS23, state sync.

## Warnings (follow-up hardening)

- **[a remotely-crashable panic is blocked only by a guard whose comment says it can't happen]** [`tm2/pkg/store/bptree/store.go:487-495`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/store/bptree/store.go#L487-L495) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/bptree/store.go#L487-L495)
  <details><summary>details</summary>

  A compressed (or batch) ICS23 proof IS wire-decodable: `BptreeCommitmentOpDecoder` `Unmarshal`s it into a `CommitmentProof_Compressed`, and that proof's `Calculate()` panics with an out-of-range index inside `cosmos/ics23/go`'s `decompressExist` (an attacker-controlled `Path` step indexes an empty lookup table). The only reason `Run` doesn't crash is the guard `if GetExist()==nil && GetNonexist()==nil`, which happens to reject the compressed/batch shapes. But the guard's comment calls itself "Not wire-reachable (Unmarshal always allocates the inner); defense-in-depth" — that rationale is false (it holds for the exist/nonexist variants, not the compressed/batch oneof members), so a future cleanup that trusts `Unmarshal` would delete the sole barrier and reintroduce a one-query node crash for any verifier wired to the registered `ics23:bptree` decoder. Fix: keep the guard, correct the comment to state it is the load-bearing barrier against compressed/batch proofs whose ics23 `Calculate` path panics, and make intent structural with a positive allow-list (`switch proof.Proof.(type)` accepting only `*_Exist`/`*_Nonexist`).
  </details>

- **[an importer with no resource bound is a state-sync OOM once it is wired]** [`tm2/pkg/bptree/import.go:103-141`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/import.go#L103-L141) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/import.go#L103-L141)
  <details><summary>details</summary>

  `Importer.Add` for a leaf entry (`Height==0`) appends to `kvBuffer` and stages a value to the DB with no per-entry cap; the only count check (`len(kvBuffer)==numKeys`, `numKeys ≤ B`) fires when a boundary marker arrives. A peer streaming leaf entries with no marker grows memory and DB writes without limit; a run of leaf+marker pairs with no inner marker pins one stack entry per leaf; and `node.Value` has no size cap. The repo has no ABCI snapshot machinery today, so `Importer`'s only callers are its own tests — this is latent, not live, but the doc comments name it the state-sync surface, so it goes hot the moment state-sync is connected to a peer. Fix: cap `len(kvBuffer)` at `B` between markers, bound `len(stack)` against the trusted target's height/size, and cap `len(node.Value)`; reject past the cap rather than buffer.
  </details>

- **[a malicious snapshot can hand a node a valid-rooted tree that halts it later]** [`tm2/pkg/bptree/node.go:94-103`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/node.go#L94-L103) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/node.go#L94-L103)
  <details><summary>details</summary>

  Inner-node separator keys route searches but are not in the Merkle commitment: `InnerNode.RebuildMiniMerkle` hashes only `childHashes`, never `keys`. Import's separator check ([`import.go:224-230`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/import.go#L224-L230) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/import.go#L224-L230)) accepts any separator inside its window `(max(left), min(right)]`, so a producer can move a separator within that window and commit a byte-identical root. The synced node passes its app-hash check at sync time, then mis-routes the first key that falls in the moved gap and diverges — a delayed self-halt (liveness), not a silent permanent fork or a forged read. Not reachable for an honest validator (it replays an ordered tx list and never an export stream; an honest Exporter emits the real separators). Fix: make the moved-separator stream rejectable at sync time — bind separators into the commitment, or have Import require the canonical form (`separator == min(right subtree)`) an honest split/Exporter always produces, instead of merely in-window.
  </details>

- **[bptree crashes on an empty key where IAVL silently commits it — a cutover hazard]** [`tm2/pkg/store/bptree/store.go:230-236`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/store/bptree/store.go#L230-L236) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/bptree/store.go#L230-L236)
  <details><summary>details</summary>

  The bptree store's `Set` validates only the value, then `MutableTree.Set` rejects an empty key with `ErrEmptyKey` ([`mutable_tree.go:104`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/mutable_tree.go#L104) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/mutable_tree.go#L104)), which the store turns into a `panic`. The incumbent IAVL store's `Set` ([`tm2/pkg/store/iavl/store.go:205`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/store/iavl/store.go#L205) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/iavl/store.go#L205)) also validates only the value and accepts an empty key as a valid leaf. So a write that IAVL committed across history would crash a bptree-backed node deterministically on replay. Practical reachability is low (gno state keys are pkgpath/object identifiers, never empty), and the panic-on-error contract itself matches IAVL — the only delta is which keys the underlying tree rejects. Fix: before mounting bptree, decide whether empty keys are legal and either make the two trees agree or confirm no historical or live store write uses an empty key. (The oversized-key cap that also panics is correct and should stay.)
  </details>

## Nits

- [`tm2/pkg/bptree/node.go:269`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/node.go#L269) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/node.go#L269) and [`:318`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/node.go#L318) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/node.go#L318) — `numKeys` is read as a uvarint then narrowed `int16(numKeys)` before the range check, so a crafted `65536` wraps to `0` and is accepted as an empty node. Not reachable from untrusted input (`ReadNode`'s only caller is `GetNode` on CRC-verified bytes; import never reaches it), so a robustness nit only. Reject `numKeys > B` on the uint64 before narrowing.

- [`tm2/pkg/bptree/remove.go:211-213`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/remove.go#L211-L213) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/remove.go#L211-L213) (and `:284-286`, `:361-365`) — leaf-branch redistribute/merge move a key between siblings by slice-header assignment without `copyKey`, unlike the inner-node branches at the same sites which copy. Currently safe (keys are never mutated in place, only replaced by shifting, per [`node.go:116-120`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/node.go#L116-L120) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/node.go#L116-L120)), but the asymmetry is a footgun for any future in-place key edit. Copy on every parent↔child key transfer for consistency.

- [`tm2/pkg/store/bptree/store.go:33`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/store/bptree/store.go#L33) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/bptree/store.go#L33) — `FastIndexEnabled` is an exported, unsynchronized, runtime-mutable package global, the shape the catalog's "global mutable state" class warns against. The "set once before mount" contract is safe but unenforced and `-race`-detectable if toggled live. Make it a `sync/atomic.Bool`, set it once at init, or thread it through `types.StoreOptions`.

- [`tm2/pkg/store/bptree/store.go:65`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/store/bptree/store.go#L65) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/bptree/store.go#L65) — `UnsafeNewStore` is exported with no doc comment stating what the caller must uphold (tree ownership / lifecycle / not shared). Document the invariant so misuse that breaks snapshot isolation is at least visible.

- [`tm2/pkg/bptree/tree_test.go:960`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/tree_test.go#L960) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/tree_test.go#L960) — `TestHashDivergence_DifferentInsertionOrder` logs on both branches and asserts nothing, so it protects against no regression. Either delete it or convert it to assert the real invariant (same op-sequence → same root, already covered elsewhere).

## Missing Tests

- **[untrusted-proof verification path has no malformed-wire fuzz]** [`tm2/pkg/store/bptree/store.go:487`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/store/bptree/store.go#L487) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/bptree/store.go#L487), [`:523`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/store/bptree/store.go#L523) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/bptree/store.go#L523)
  <details><summary>details</summary>

  `TestStore_ProofDecoder` checks only registration ("we can't easily test decoding"). The decoder + `Run` are the consensus-adjacent untrusted-input surface (the registered `ics23:bptree` op runs attacker bytes). Add a fuzz target feeding random/truncated bytes to `BptreeCommitmentOpDecoder` (must error, never panic) and decoded-but-bogus proofs to `Run` with 0/1/2 args (must error, never return a root). This is the regression guard for Warning 1.
  </details>

- **[no proof-forgery negative test tampering the proof path or key]** [`tm2/pkg/bptree/proof.go:194`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/proof.go#L194) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/proof.go#L194)
  <details><summary>details</summary>

  Existing negatives cover wrong-value, wrong-root, and nil-inner, but none flips a byte of a valid proof's `InnerOp` prefix/suffix or swaps `exist.Key`. ics23 rejects both today; a test pins it so a future loosening of `BptreeSpec` (depth bounds, domain separators) can't silently make the chain forgeable. Confirmed behaviorally: tampering one InnerOp suffix byte or the proof key makes `VerifyMembership` return false.
  </details>

- **[import streaming-Add has no resource-bound test]** [`tm2/pkg/bptree/import.go:103`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/import.go#L103) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/import.go#L103)
  <details><summary>details</summary>

  Coverage streams one leaf then commits; nothing exercises many `Height==0` entries with no marker. Add a test asserting `Add` rejects (or caps) once the inter-marker buffer exceeds `B`. This is the regression guard for Warning 2 and the place to confirm any future state-sync wiring keeps the bound.
  </details>

## Suggestions

- Warnings 1–3 are the natural contents of one small follow-up hardening PR, all behind "before bptree is mounted / state-sync is wired." None blocks anything today.

- The compressed-proof crash (Warning 1) lives in `cosmos/ics23/go` and is shared with IAVL: any future caller that forwards a compressed/batch proof to `ics23.Verify*`/`Calculate()` crashes. Worth a one-line note in the follow-up that the registered-decoder pattern must never forward compressed/batch proofs to ics23, plus consideration of an upstream bounds fix.

## Verified sound (deep negatives)

The strong negative results, each backed by runnable tests during this review:

- **Proof soundness.** No forgery: value-swap, key-swap, claimed-value mismatch, cross-version reuse (v1 proof vs v2 root), and cross-tree reuse all rejected; non-membership cannot be produced for an existing key, and widened-gap forgeries that hide a real key (including across leaf-node boundaries), fake-leftmost/rightmost forgeries, and over-`MaxDepth`/truncated/padded/spliced paths all fail. Empty-valued membership proofs generate but never verify (ics23 rejects empty values), with no soundness gap. A 1.3M-exec native fuzz plus ~40k mutated/random inputs through the registered decoder + `Run` produced zero panics. Root binding holds: the outer `merkle.ProofOperators.Verify` ([`tm2/pkg/crypto/merkle/proof.go:60`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/crypto/merkle/proof.go#L60) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/crypto/merkle/proof.go#L60)) checks the proof-computed root against the trusted root.

- **Determinism / no consensus fork.** The root is a pure function of the op sequence; identical sequences (including restart-then-continue vs in-memory) give identical roots. The B+ shape is insertion-order-dependent, but the cache store sorts each block's write set (`sort.Strings` at [`tm2/pkg/store/cache/store.go:268`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/store/cache/store.go#L268) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/store/cache/store.go#L268)) before it reaches the tree, so write order never reaches the tree unsorted on the deliver path. No map-iteration / time / rand / float / pointer-identity feeds any hashed or serialized field (NodeKeys/nonces are not hashed). Export→Import reproduces the exact root across single-key, deletion-degenerate, empty-value, prefix-key, and large-key/value trees. Forward+reverse iteration is strictly sorted, complete, and duplicate-free over thousands of keys after deletes, and a 20k-op differential fuzz matches a reference map and is bit-identical across runs.

- **Pruning / version state.** The dual-tree-walk never deletes a record a retained version still references (40-seed random churn, a height-≥3 deletion-heavy tall tree forcing inner merges/redistributes, and gap/non-contiguous prune ranges, each fully verifying every retained version by point-read and iteration); orphan values neither leak nor get wrongly deleted (post-full-prune the value-record count collapses to exactly the live set). The round-1 bugs are fixed: idempotent `SaveVersion` then `Rollback` preserves persisted values; `versionExistsE` propagates DB errors; `getChild` returns an error rather than panicking on not-found; the fast index never diverges from the authoritative tree across Set/Remove/overwrite/prune/rebuild (the `vkVersion ≤ snapshot` gate is sufficient). Concurrent prune + reader is safe: `ImmutableTree.Iterator` and `NewIteratorWithNDB` now register version readers ([`iterator.go:391`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/iterator.go#L391) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/iterator.go#L391), [`:431`](https://github.com/gnolang/gno/blob/51892161f/tm2/pkg/bptree/iterator.go#L431) · [↗](../../../../../.worktrees/gno-review-5438/tm2/pkg/bptree/iterator.go#L431)) and `beginPruning` rejects pruning a range with an active reader, verified under `-race`.

- **Memory safety.** Every slice handed to a caller is copied — `MutableTree.Get`, `Iterator.Key()`/`Value()`, `fastGet`'s `payload[8:]`, and the by-index/resolved walks — so mutating a returned slice (including append-past-cap then in-place overwrite) leaves committed state, the node cache, and the app hash intact. The store's query path returns graceful errors (never panics) on empty/huge keys, negative/huge heights, and Prove on absent/oversized keys; the iterator's panic-on-unacknowledged-error never fires on clean iteration of a healthy tree.

## Open questions

- What is the cutover plan and timeline for actually mounting bptree (genesis flag, per-store config, hardfork)? Every Warning's severity is gated on it; nothing here is urgent while production stays on IAVL. Not posted — a roadmap question, not a code change.
- Does any historical or live gno store write ever use an empty key? If provably never, Warning 4 downgrades to a documented divergence. Not posted — answerable from genesis/state inspection, not from this diff.

## Repros

All four Warning repros are self-contained, run from a fresh gnolang/gno clone at the merged commit, and were executed during this review.

<details><summary>W1 — compressed proof decodes, panics in Calculate(), blocked only by the Run guard</summary>

```bash
# from a local clone of gnolang/gno (code merged to master at 51892161f):
git fetch origin master && git checkout 51892161f
mkdir -p tm2/pkg/bptree/zzrepro
cat > tm2/pkg/bptree/zzrepro/w1_test.go <<'EOF'
package zzrepro

import (
	"testing"

	ics23 "github.com/cosmos/ics23/go"
	bp "github.com/gnolang/gno/tm2/pkg/bptree"
	storebp "github.com/gnolang/gno/tm2/pkg/store/bptree"
)

func TestW1(t *testing.T) {
	ce := &ics23.CompressedExistenceProof{Key: []byte("k"), Value: []byte("v"),
		Leaf: bp.BptreeSpec.LeafSpec, Path: []int32{7}} // indexes an empty lookup table
	cbp := &ics23.CompressedBatchProof{
		Entries:      []*ics23.CompressedBatchEntry{{Proof: &ics23.CompressedBatchEntry_Exist{Exist: ce}}},
		LookupInners: nil,
	}
	proof := &ics23.CommitmentProof{Proof: &ics23.CommitmentProof_Compressed{Compressed: cbp}}
	pop := storebp.NewBptreeCommitmentOp([]byte("k"), proof).ProofOp() // marshals to wire bytes
	decoded, err := storebp.BptreeCommitmentOpDecoder(pop)
	if err != nil {
		t.Fatalf("decoder rejected compressed proof (unexpected): %v", err)
	}
	co := decoded.(storebp.CommitmentOp)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Calculate() PANIC on decoded compressed proof: %v", r)
			} else {
				t.Fatal("expected Calculate() panic")
			}
		}()
		_, _ = co.Proof.Calculate()
	}()
	if _, rerr := decoded.Run([][]byte{[]byte("v")}); rerr == nil {
		t.Fatal("expected Run guard error")
	} else {
		t.Logf("Run() guard converted it to an error instead of crashing: %v", rerr)
	}
}
EOF
go test -run TestW1 -v ./tm2/pkg/bptree/zzrepro/
rm -rf tm2/pkg/bptree/zzrepro
```

```
=== RUN   TestW1
    w1_test.go:27: Calculate() PANIC on decoded compressed proof: runtime error: index out of range [7] with length 0
    w1_test.go:38: Run() guard converted it to an error instead of crashing: proof is neither an existence nor a non-existence proof
--- PASS: TestW1
```
</details>

<details><summary>W2 — importer accepts unbounded leaf entries with no boundary marker</summary>

```bash
# from a local clone of gnolang/gno (code merged to master at 51892161f):
git fetch origin master && git checkout 51892161f
mkdir -p tm2/pkg/bptree/zzrepro
cat > tm2/pkg/bptree/zzrepro/w2_test.go <<'EOF'
package zzrepro

import (
	"fmt"
	"testing"

	bp "github.com/gnolang/gno/tm2/pkg/bptree"
	memdb "github.com/gnolang/gno/tm2/pkg/db/memdb"
)

func TestW2(t *testing.T) {
	tree := bp.NewMutableTreeWithDB(memdb.NewMemDB(), 1000, bp.NewNopLogger())
	imp, err := tree.Import(1)
	if err != nil {
		t.Fatal(err)
	}
	defer imp.Close()
	const n = 200_000
	for i := 0; i < n; i++ {
		if err := imp.Add(&bp.ExportNode{Key: []byte(fmt.Sprintf("k%07d", i)), Value: []byte("v"), Height: 0}); err != nil {
			t.Fatalf("Add rejected at %d: %v (a cap would surface here)", i, err)
		}
	}
	t.Logf("accepted %d leaf entries with no boundary marker and no rejection (unbounded buffer + %d staged value writes)", n, n)
}
EOF
go test -run TestW2 -v ./tm2/pkg/bptree/zzrepro/
rm -rf tm2/pkg/bptree/zzrepro
```

```
=== RUN   TestW2
    w2_test.go:24: accepted 200000 leaf entries with no boundary marker and no rejection (unbounded buffer + 200000 staged value writes)
--- PASS: TestW2
```
</details>

<details><summary>W3 — moved-within-window separator imports to the identical root</summary>

```bash
# from a local clone of gnolang/gno (code merged to master at 51892161f):
git fetch origin master && git checkout 51892161f
mkdir -p tm2/pkg/bptree/zzrepro
cat > tm2/pkg/bptree/zzrepro/w3_test.go <<'EOF'
package zzrepro

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	bp "github.com/gnolang/gno/tm2/pkg/bptree"
	memdb "github.com/gnolang/gno/tm2/pkg/db/memdb"
)

func TestW3(t *testing.T) {
	tree := bp.NewMutableTreeWithDB(memdb.NewMemDB(), 1000, bp.NewNopLogger())
	for i := 0; i < 200; i++ {
		if _, err := tree.Set([]byte(fmt.Sprintf("k%04d", i)), []byte("v")); err != nil {
			t.Fatal(err)
		}
	}
	h0, v, err := tree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}
	imm, err := tree.GetImmutable(v)
	if err != nil {
		t.Fatal(err)
	}
	defer imm.Close()
	imm.SetValueResolver(tree.GetCommittedValueByKey)
	exp, err := imm.Export(nil)
	if err != nil {
		t.Fatal(err)
	}
	var nodes []*bp.ExportNode
	var leafKeys [][]byte
	for {
		nd, err := exp.Next()
		if errors.Is(err, bp.ErrExportDone) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, nd)
		if nd.Height == 0 {
			leafKeys = append(leafKeys, nd.Key)
		}
	}
	exp.Close()
	for _, nd := range nodes {
		if nd.Height > 0 && len(nd.SeparatorKeys) > 0 {
			sep := nd.SeparatorKeys[0]
			var prev []byte
			for _, k := range leafKeys {
				if bytes.Compare(k, sep) < 0 {
					prev = k
				} else {
					break
				}
			}
			alt := append(append([]byte{}, prev...), 0x01) // strictly inside (prev, sep)
			if prev != nil && bytes.Compare(alt, sep) < 0 {
				nd.SeparatorKeys[0] = alt
				t.Logf("moved separator %q -> %q (window (%q, %q])", sep, alt, prev, sep)
				break
			}
		}
	}
	tree2 := bp.NewMutableTreeWithDB(memdb.NewMemDB(), 1000, bp.NewNopLogger())
	imp, err := tree2.Import(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, nd := range nodes {
		if err := imp.Add(nd); err != nil {
			t.Fatalf("import rejected the moved-separator stream: %v", err)
		}
	}
	if err := imp.Commit(); err != nil {
		t.Fatal(err)
	}
	imp.Close()
	if h1 := tree2.Hash(); !bytes.Equal(h0, h1) {
		t.Fatalf("roots differ (%x vs %x) — separator IS committed", h0, h1)
	}
	t.Logf("moved-separator stream accepted with identical root %x — separators are outside the commitment", h0)
}
EOF
go test -run TestW3 -v ./tm2/pkg/bptree/zzrepro/
rm -rf tm2/pkg/bptree/zzrepro
```

```
=== RUN   TestW3
    w3_test.go:65: moved separator "k0031" -> "k0030\x01" (window ("k0030", "k0031"])
    w3_test.go:90: moved-separator stream accepted with identical root 5655587a…cd60eca — separators are outside the commitment
--- PASS: TestW3
```
</details>

<details><summary>W4 — store panics on an empty key (IAVL accepts it)</summary>

```bash
# from a local clone of gnolang/gno (code merged to master at 51892161f):
git fetch origin master && git checkout 51892161f
mkdir -p tm2/pkg/bptree/zzrepro
cat > tm2/pkg/bptree/zzrepro/w4_test.go <<'EOF'
package zzrepro

import (
	"testing"

	memdb "github.com/gnolang/gno/tm2/pkg/db/memdb"
	storebp "github.com/gnolang/gno/tm2/pkg/store/bptree"
	types "github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestW4(t *testing.T) {
	st := storebp.StoreConstructor(memdb.NewMemDB(), types.StoreOptions{})
	defer func() {
		if r := recover(); r != nil {
			t.Logf("store.Set(emptyKey, v) PANIC (IAVL would accept it): %v", r)
		} else {
			t.Fatal("expected panic on empty key")
		}
	}()
	st.(types.Store).Set(nil, []byte{}, []byte("v"))
}
EOF
go test -run TestW4 -v ./tm2/pkg/bptree/zzrepro/
rm -rf tm2/pkg/bptree/zzrepro
```

```
=== RUN   TestW4
    w4_test.go:16: store.Set(emptyKey, v) PANIC (IAVL would accept it): key must not be empty
--- PASS: TestW4
```
</details>
