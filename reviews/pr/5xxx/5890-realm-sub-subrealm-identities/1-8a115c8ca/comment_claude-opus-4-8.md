# Review: PR [#5890](https://github.com/gnolang/gno/pull/5890)
Event: APPROVE

## Body
Looks good. Verified on 8a115c8ca: removing the `if !rlm.IsCurrent()` panic in [NewBanker](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/stdlibs/chain/banker/banker.gno#L101) [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/banker/banker.gno#L101) lets the malicious realm's `Attack` tx succeed and move the victim's coins, so the gate closes a real caller-drain.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5890 -R gnolang/gno

# Disable NewBanker's live-cur gate.
sed -i 's/if !rlm.IsCurrent() {/if false \&\& !rlm.IsCurrent() {/' \
  gnovm/stdlibs/chain/banker/banker.gno

# TEST 6: the victim signs a call into a malicious realm that builds a
# RealmSend banker over cur.Previous() (the victim) and sends its coins.
go test ./gno.land/pkg/integration/ -run 'TestTestdata/banker_security' -count=1

git checkout -- gnovm/stdlibs/chain/banker/banker.gno
```

```
> ! gnokey maketx call -pkgpath gno.land/r/test/prevattack -func Attack ...
OK!
GAS USED:   1672221
FAIL: testdata/banker_security.txtar:97: unexpected "gnokey" command success
FAIL: testdata/banker_security.txtar:98: no match for `banker can only be instantiated for the current realm` found in stderr
FAIL
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5890-realm-sub-subrealm-identities/1-8a115c8ca/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
