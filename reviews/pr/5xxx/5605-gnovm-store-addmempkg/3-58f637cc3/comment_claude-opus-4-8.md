# Review: PR #5605
Event: APPROVE

## Body
Sound fix. Verified on 58f637cc3 that the reorder writes the same key/value multiset as the deleted counter-first path, so committed state and per-tx gas are byte-identical and the change cannot fork consensus, and that the "substore divergence" panic is live on the production restart path: the restart store sets no `pkgGetter` ([keeper.go:149](https://github.com/gnolang/gno/blob/58f637cc3/gno.land/pkg/sdk/vm/keeper.go#L149)), so a body-less index slot panics at boot rather than being silently re-fabricated.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5605-gnovm-store-addmempkg/3-58f637cc3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gnovm/pkg/gnolang/store.go:961-976 [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L961)
This comment lays out a per-statement crash taxonomy ("orphaned body is harmless", "self-heals on retry") and then says cross-substore flush ordering is non-deterministic, which cancels it: the base and iavl substores commit in non-deterministic order at block commit, so the index can become durable before the body regardless of statement order, and the real safety net is IterMemPackage's fail-fast panic, not the ordering. It also says a consumer-side nil skip "must be retained as belt-and-braces," but the only consumer ([machine.go:329](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/machine.go#L329)) has no nil guard, and the cited commit `b15ffde6e` is not reachable from a gnolang/gno clone. Rewrite it to state the best-effort-ordering-plus-fail-fast-panic posture and drop the nil-skip and commit references.

*(AI Agent)*

## gnovm/pkg/gnolang/store_test.go:283-332 [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L283)
TestAddMemPackage_WriteOrderIsBodyFirst checks only the final state, not the Set call order, so a regression back to the old counter→index→body order still passes it (and the "Verified by snapshotting each substore" comment describes snapshotting the test never does). Record the Set sequence and assert body→index→counter; an adversarial recorder test that does this is in the repro.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5605 -R gnolang/gno
curl -fsSL -o gnovm/pkg/gnolang/addmempkg_write_order_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5605-gnovm-store-addmempkg/3-58f637cc3/tests/addmempkg_write_order_test.go
# flip AddMemPackage back to the old counter->index->body order
python3 - <<'PY'
p="gnovm/pkg/gnolang/store.go"; s=open(p).read()
body='\tds.iavlStore.Set(ds.gctx, pathkey, bz)\n'
ctr='\tds.baseStore.Set(ds.gctx, ctrkey, []byte(strconv.FormatUint(ctr, 10)))\n'
s=s.replace(body,'',1); s=s.replace(ctr,ctr+body,1); open(p,"w").write(s)
PY
echo "--- shipped test under the OLD order:"
go test -count=1 -run 'TestAddMemPackage_WriteOrderIsBodyFirst$' ./gnovm/pkg/gnolang/
echo "--- adversarial recorder test under the OLD order:"
go test -count=1 -run 'TestAdv_AddMemPackage_RecordedWriteOrder$' ./gnovm/pkg/gnolang/
git checkout -- gnovm/pkg/gnolang/store.go
rm gnovm/pkg/gnolang/addmempkg_write_order_test.go
```

```
--- shipped test under the OLD order:
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.010s
--- adversarial recorder test under the OLD order:
--- FAIL: TestAdv_AddMemPackage_RecordedWriteOrder
    REGRESSION: body (iavl pkg:) must be written before index (base pkgidx:N) — ops=[{base pkgidx:00000000000000000001} {base pkgidx:counter} {iavl pkg:_/ord}]
FAIL
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	0.009s
```
</details>

*(AI Agent)*

## gnovm/pkg/gnolang/store.go:967-970 [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L967)
This comment promises that after a crash between the index write and the counter bump, the next AddMemPackage recomputes `ctr = prevCtr+1` and overwrites the dangling slot, but no test pins this self-healing path. Add one: forge an index slot at N+1 with the counter left at N, call AddMemPackage again, and assert the slot was overwritten with the new path and the counter is N+1 (not N+2).

*(AI Agent)*
