# Review: [#6058](https://github.com/gnolang/gno/pull/6058)
Event: APPROVE
Verdict: APPROVE. One Nit stays open: a blank type switch variable reaches `defineFauxCopy` with a slot it never got, which only the REPL and expression parsing hit.
Model: claude-opus-5-5, standard review, the finder at xhigh and the judge at high
Commit: a40c1fbf7
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6058 a40c1fbf7`
Round: 2. The head moved from 391840aa7 to a40c1fbf7, a merge of master that carried in #6056 and one commit answering round 1. Six of round 1's seven threads are fixed there: the type switch test, the last-wins unit test, the dead `NSTypeSwitch` branch, the unconditional `defineFauxCopy` check and both ADR paragraphs. The seventh, the accounted allocation of a `fallthrough` chain, left this diff with #6056 and stays open. One finder, a reflector pass by the judge, 2 candidates, each run or read from scratch by an agent that was not its finder; the finder settled 9 more with the line that cleared each.

## Body

- Unresolved: [accounted allocation grows with the length of a `fallthrough` chain](https://github.com/gnolang/gno/pull/6058#discussion_r3863669975).

## gnovm/pkg/gnolang/preprocess.go:1057 [gh](https://github.com/gnolang/gno/blob/a40c1fbf7/gnovm/pkg/gnolang/preprocess.go#L1057) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/preprocess.go#L1057) · Nit
Nit: a blank type switch variable gets no slot, so the REPL fails `switch _ := i.(type)` here with an internal error instead of Go's `no new variables`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6058 -R gnolang/gno
curl -fsSL -o gnovm/pkg/repl/zz_blank_tsvar_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6058-switch-case-shadowing/2-a40c1fbf7/tests/judge-1-blank-tsvar-repl_test.go
go test -count=1 -v -run TestJudgeBlankTSVar ./gnovm/pkg/repl/
rm gnovm/pkg/repl/zz_blank_tsvar_test.go
```

The first two statements fail on the internal slot check; on the merge base they run, and the third fails there too, at runtime:

```
"switch _ := i.(type) { case int: x := 3; println(\"blank-body\", x) }"
  err=":1:31-73: faux copy slot 0 holds x, not _"
"switch _ := i.(type) { case int: }; println(\"blank-empty ok\")"
  err="runtime error: index out of range [0] with length 0"
"switch _ := i.(type) { case int: println(\"blank-stmt\") }"
  err="runtime error: index out of range [0] with length 0"
"switch y := i.(type) { case int: x := 3; println(\"named\", x, y) }"
  out="named 3 42"
```

A file never gets here: `ParseFile` parses with `DeclarationErrors`, and the parser's resolver rejects the statement first. `ParseStmts` (the REPL) and `ParseExpr` (`gno run -expr`, `qeval`) parse without that check; both recover the panic into an error.

</details>
