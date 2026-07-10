# Review: PR [#5933](https://github.com/gnolang/gno/pull/5933)
Event: APPROVE

## Body
Looks good, verified on 61f762f9b. Restoring the old decode-into-a-zero-valued-struct plus mergo.Merge load path reproduces the bug against the PR's own tests: consensus.create_empty_blocks and mempool.recheck load back as true, and rpc.cors_allowed_methods = [] refills to its four defaults.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5933 -R gnolang/gno
fix=$(git log --grep='honor explicit zero values in node config loader' --format=%H -1)
git checkout "$fix"^ -- tm2/pkg/bft/config/config.go tm2/pkg/bft/config/toml.go go.mod go.sum
go test ./tm2/pkg/bft/config/ -run 'TestConfig_LoadConfig/explicit' -v
git checkout HEAD -- tm2/pkg/bft/config/config.go tm2/pkg/bft/config/toml.go go.mod go.sum
```

```
Error: Should be false
Error: Should be empty, but was [HEAD GET POST OPTIONS]
--- FAIL: TestConfig_LoadConfig/explicit_zero_values_are_honored
--- FAIL: TestConfig_LoadConfig/explicit_empty_slice_is_honored
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5933-honor-explicit-zero-config/1-61f762f9b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
