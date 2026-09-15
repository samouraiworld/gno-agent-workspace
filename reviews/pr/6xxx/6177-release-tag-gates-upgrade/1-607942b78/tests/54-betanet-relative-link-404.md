# Candidate 54: docs/resources/gnoland-networks.md:8 — relative link 404s on the published docs site

Browser-reachable surface, no test harness (per Repro rules: "A finding on a
surface the reader reaches in a browser ships no harness in the comment").
This file documents the check that was actually run.

## Diff

```
-| Betanet (retired) | —                                       | `gnoland1`  | [`misc/deployments/gnoland1`](https://github.com/gnolang/gno/tree/chain/gnoland1/misc/deployments/gnoland1)                |
+| Betanet (retired) | —                                       | `gnoland1`  | [`misc/deployments/betanet`](../../misc/deployments/betanet)                                                               |
```

Mainnet and Pearl rows in the same table keep full
`https://github.com/gnolang/gno/tree/...` URLs.

## Check

1. `.github/workflows/deploy-docs.yml` triggers on `docs/**` and dispatches
   `netlify.yml` in `gnolang/docs.gno.land`, a Docusaurus build of the `docs/`
   tree only; `misc/` is not part of that site.
2. The live page already carries a relative link of the same shape, pre-existing
   on the Staging row (`[misc/loop](../../misc/loop)`, unchanged by this PR).
   Fetching the rendered page shows Docusaurus resolves it to a same-origin
   absolute path:

   ```
   $ WebFetch https://docs.gno.land/resources/gnoland-networks
   Staging: /misc/loop
   ```

3. Fetching that resolved path confirms it 404s today, before this PR lands:

   ```
   $ WebFetch https://docs.gno.land/misc/loop
   The server returned HTTP 404 Not Found.
   ```

4. The PR turns the Betanet row's link from a working absolute GitHub URL into
   the same broken relative shape, so it will resolve to
   `https://docs.gno.land/misc/deployments/betanet` and 404 the same way, while
   Mainnet and Pearl keep the working absolute form.

## Repro from a plain clone

```bash
# from a local clone of gnolang/gno, at the PR head:
git fetch origin pull/6177/head
git checkout FETCH_HEAD
sed -n '5,9p' docs/resources/gnoland-networks.md
```

```
| Betanet (retired) | —                                       | `gnoland1`  | [`misc/deployments/betanet`](../../misc/deployments/betanet)                                                               |
```

Then, once the docs site rebuilds off this branch, load
`https://docs.gno.land/resources/gnoland-networks` and click the Betanet
deployment-files link; compare against the Mainnet/Pearl rows' links which
stay absolute and load on GitHub.
