# Review: PR [#5907](https://github.com/gnolang/gno/pull/5907)
Event: REQUEST_CHANGES

## Body
[`grc20_registry_emit`](https://github.com/gnolang/gno/blob/fba248c95/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L14) still looks up [`foo20`](https://github.com/gnolang/gno/blob/fba248c95/examples/gno.land/r/demo/defi/foo20/foo20.gno#L25) by its bare path, but `Register` now keys tokens as `path.SYMBOL`, so `MustGet` panics `unknown token` and the `main / test` job is red. The `TransferBy` argument on line 14 needs to become `gno.land/r/demo/defi/foo20.FOO`. Reproduced on fba248c95.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5907 -R gnolang/gno
go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20_registry_emit' -count=1
```

```
    --- FAIL: TestTestdata/grc20_registry_emit (2.5s)
        Data: unknown token: gno.land/r/demo/defi/foo20
        panic: unknown token: gno.land/r/demo/defi/foo20
        MustGet at gno.land/r/demo/defi/grc20reg/grc20reg.gno:59
        TransferBy at gno.land/r/grc20transfer/grc20transfer.gno:6
        FAIL: testdata/grc20_registry_emit.txtar:14: unexpected "gnokey" command failure
FAIL
```

Changing the line 14 argument to `gno.land/r/demo/defi/foo20.FOO` turns the test green.
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5907-prevent-token-path-overwrite/1-fba248c95/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:43 [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43)
The register event still emits `slug`, but `slug` no longer affects the registry key. A caller that passes one, like [`grc20factory`](https://github.com/gnolang/gno/blob/fba248c95/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L51) passing the symbol or [`tokenhub`](https://github.com/gnolang/gno/blob/fba248c95/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L34) passing a user slug, has it silently dropped, and an indexer reading `slug` gets a value that maps to no registry entry. Drop it from the event, or document that it is ignored.
</content>
