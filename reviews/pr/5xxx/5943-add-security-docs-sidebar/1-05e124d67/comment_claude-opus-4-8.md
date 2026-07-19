# Review: PR [#5943](https://github.com/gnolang/gno/pull/5943)
Event: REQUEST_CHANGES

## Body
[`docs/resources/gno-security.md`](https://github.com/gnolang/gno/blob/05e124d67/docs/resources/gno-security.md) names its companion `SECURITY_GUIDE.md` at [line 5](https://github.com/gnolang/gno/blob/05e124d67/docs/resources/gno-security.md?plain=1#L5) and [line 45](https://github.com/gnolang/gno/blob/05e124d67/docs/resources/gno-security.md?plain=1#L45), but no such file exists. The page it means, [`gno-security-guide.md`](https://github.com/gnolang/gno/blob/05e124d67/docs/resources/gno-security-guide.md), is one this PR makes navigable, so the dangling name reaches readers on merge.

Repros run at 05e124d67.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5943-add-security-docs-sidebar/1-05e124d67/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## misc/docs/sidebar.json:48-50 [↗](../../../../../.worktrees/gno-review-5943/misc/docs/sidebar.json#L48-L50)
[`misc/docs/sidebar.json`](https://github.com/gnolang/gno/blob/05e124d67/misc/docs/sidebar.json) is generated from [`docs/README.md`](https://github.com/gnolang/gno/blob/05e124d67/docs/README.md?plain=1#L36) by [`docs/Makefile:6`](https://github.com/gnolang/gno/blob/05e124d67/docs/Makefile#L6), and the README's References list is unchanged, so the next `make -C docs generate` deletes these three lines. No workflow regenerates or diffs the file, so that drift stays silent until someone runs the target. Separately, [`deploy-docs.yml`](https://github.com/gnolang/gno/blob/05e124d67/.github/workflows/deploy-docs.yml#L8-L9) builds the site only for pushes touching `docs/**`, so this change alone never reaches the published navigation.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5943 -R gnolang/gno
go run -C misc/docs/tools/indexparser . -path "$PWD/docs/README.md" > misc/docs/sidebar.json
git diff --stat misc/docs/sidebar.json
git diff misc/docs/sidebar.json
git checkout HEAD -- misc/docs/sidebar.json
```

```
 misc/docs/sidebar.json | 3 ---
 1 file changed, 3 deletions(-)
diff --git a/misc/docs/sidebar.json b/misc/docs/sidebar.json
index 8a5b7e4da..053739fbb 100644
--- a/misc/docs/sidebar.json
+++ b/misc/docs/sidebar.json
@@ -45,9 +45,6 @@
           "resources/gno-testing",
           "resources/realms",
           "resources/gno-interrealm",
-          "resources/gno-interrealm-v2",
-          "resources/gno-security",
-          "resources/gno-security-guide",
           "resources/gno-memory-model",
```

The regenerated blob `053739fbb` is the pre-PR sidebar, the same base blob this PR's own diff header names.
</details>
