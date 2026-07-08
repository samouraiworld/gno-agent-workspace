# Review: PR [#5873](https://github.com/gnolang/gno/pull/5873)
Event: REQUEST_CHANGES

## Body
Verified on eb829bec3: rendering the testnet-URL cell reproduces the double backtick as two visible characters, where master's zero-width-space form renders as a clean inline URL. The reference anchors all resolve against the target headings, a cross-file check the docs linter skips.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5873-rewrite-gnokey-guide-reference/2-eb829bec3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/builders/getting-started.md:275 [↗](../../../../../.worktrees/gno-review-5873-h/docs/builders/getting-started.md#L275)
The testnet URL cell breaks the anti-autolink split. The opening single-backtick span closes at the trailing backtick, so the two inner backticks render as literal characters inside one code span instead of a clean inline URL. Restore the zero-width-space split, or use two separate single-backtick spans.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5873 -R gnolang/gno
python3 - <<'PY'
import markdown
line = next(l for l in open("docs/builders/getting-started.md") if "rpc.<testN>" in l)
cell = line.split("|")[3].strip()
print(markdown.markdown(cell))
PY
```

```
<p><code>https://``rpc.&lt;testN&gt;.testnets.gno.land:443</code></p>
```
</details>

## docs/users/using-gnokey.md:141 [↗](../../../../../.worktrees/gno-review-5873-h/docs/users/using-gnokey.md#L141)
"Calling `Deposit` on the `wugnot` realm to wrap `1000ugnot`." has no main verb. Fold it into the surrounding sentence or make it a full one.

## docs/builders/getting-started.md:369 [↗](../../../../../.worktrees/gno-review-5873-h/docs/builders/getting-started.md#L369)
The link text reads "`addpkg` in Interact with gnokey" but the target is the page now titled "gnokey command reference". The same "Interacting with gnokey" label sits over reference-page anchors in query-state-api.md:6 and :210, rpc-clients.md:29, and gno-packages.md:49. The links resolve; only the label is stale.
