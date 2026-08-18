Corrected text for the open thread https://github.com/gnolang/gno/pull/5981#discussion_r3660033767, whose posted body counts three forms and is now four. Apply with:

```bash
gh api -X PATCH repos/gnolang/gno/pulls/comments/3660033767 --field body="$(sed -n '/^---8<---$/,$p' thread_r3660033767_edit.md | tail -n +2)"
```

---8<---
`func f(iota int) { println("hi") }`, `func f() (iota int)`, `func (iota T) M()` and `for iota := 0; iota < 2; iota++` all run on master, the first three because the name is bound but never [referenced](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L1304) and the fourth because the `.loopvar` rename [rewrites its body references](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/preprocess.go#L372) to match. Node startup [re-preprocesses](https://github.com/gnolang/gno/blob/055d85cbc/gnovm/pkg/gnolang/machine.go#L356-L361) every stored package at [`VMKeeper.Initialize`](https://github.com/gnolang/gno/blob/055d85cbc/gno.land/pkg/sdk/vm/keeper.go#L162) with no per-package recover, so a package already on chain that uses one of those forms fails at boot rather than at its next call. Nothing under `examples/` or `gnovm/stdlibs/` binds `iota`, so the exposure is third-party packages.
