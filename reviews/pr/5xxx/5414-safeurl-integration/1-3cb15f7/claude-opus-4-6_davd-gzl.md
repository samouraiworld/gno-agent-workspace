# PR #5414: feat(gnoweb): add SafeURL integration for external URL validation

**URL:** https://github.com/gnolang/gno/pull/5414
**Author:** alanrsoares | **Base:** master | **Files:** 43 | **+1997 -39**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR integrates a SafeURL API into gnoweb to validate external URLs and images in markdown content before rendering. The feature is opt-in, disabled by default, and activated via `--safeurl-api-key` and `--safeurl-url` CLI flags.

**Architecture:**
1. **Server-side** (`safeurl/` package): A `Validator` wraps a `Client` which calls the SafeURL SDK to batch-scan external URLs. Results are cached in an LRU cache with TTL. The `Client` uses the `gnoverse/safeurl-sdk` for API calls.
2. **Markdown pipeline** (`markdown/ext_safeurl.go`): A Goldmark AST transformer (priority 400) collects all external URLs from links and images, batch-validates them via the validator, stores results in the parser context, and wraps `ast.Image` nodes in `SafeImage` with safety status. The existing link transformer (priority 500) reads safety results from the parser context and annotates `GnoLink` nodes.
3. **Rendering**: Links and images are rendered in different states: `safe` (normal rendering with SafeURL info in title), `unsafe` (links: clickable with warning icon; images: blocked), `pending` (links: non-clickable span with copy button and spinner; images: placeholder with spinner), `unavailable` (plain text, not clickable).
4. **Client-side polling** (`controller-safeurl.ts`): A TypeScript controller polls `/api/safeurl/scan/{id}` for pending scan results and updates the DOM when scans complete, transitioning elements from pending to their final state.
5. **Controller minification fix**: All controllers gain a `static controllerIdentifier` property to survive JS minification, as class names are mangled during build.

**Other changes:** Go module dependencies updated (`safeurl-sdk`, `oapi-codegen/runtime`, `go-jsonmerge`), CSS styles for safety indicators, a loading spinner icon, and the article template now includes the `safeurl` data-controller.

CI has not run full checks — "Merge Requirements" shows "Some requirements are not satisfied yet."

## Test Results
- **Existing tests:** PASS — `safeurl` package tests all pass (cache + validator tests)
- **Edge-case tests:** 1 written — `adversarial_isexternalurl_test.go` confirms 6/6 `IsExternalURL` bypass scenarios (see Critical #1)

## Critical (must fix)
- [ ] `validator.go:163` — **Security bypass in `IsExternalURL`**: Uses `strings.Contains(lowerURL, "gno.land")` to classify URLs as internal. This is a substring match, not a domain match. Confirmed by adversarial test: `https://evil.com/gno.land/malware`, `https://gno.land.evil.com/`, `https://notgno.land/something`, `https://evil.com?redirect=gno.land`, `https://evil.com#gno.land`, and `https://gno.land@evil.com/` all bypass scanning and are treated as "internal" (safe). An attacker can craft a malicious URL containing "gno.land" anywhere to evade safety checks. **Fix:** Parse the URL and check the hostname with proper suffix matching:
  ```go
  parsed, err := url.Parse(rawURL)
  if err != nil { return true }
  host := parsed.Hostname()
  return host != "gno.land" && !strings.HasSuffix(host, ".gno.land")
  ```

- [ ] `controller-safeurl.ts:229-232` — **XSS vulnerability**: `showUnsafeImageWarning()` uses `innerHTML` with user-controlled `alt` text: `` element.innerHTML = `...Unsafe image blocked: ${alt}...` ``. The `alt` value comes from `getAttribute("data-safeurl-alt")` which returns decoded HTML entities. If alt text contains `<img onerror=alert(1)>`, it would execute. **Fix:** Use `textContent` on a created element instead of string interpolation in `innerHTML`.

- [ ] `main.go:377` — **CSP blocks SafeURL polling**: The `connect-src` directive is `connect-src %s/abci_query` (only the RPC node). The SafeURL polling endpoint `/api/safeurl/scan/...` is same-origin, but `'self'` is not in `connect-src`. In strict mode, the browser will block `fetch('/api/safeurl/scan/...')` calls, making the entire client-side polling feature non-functional. **Fix:** Add `'self'` to connect-src: `connect-src 'self' %s/abci_query`.

## Warnings (should fix)
- [ ] `cache.go:42-63` — **Race condition in `Get()`**: Acquires `RLock` to read the entry (line 42-44), releases it, then acquires `Lock` to remove expired or move-to-front (lines 52-54, 59-61). Between releasing RLock and acquiring Lock, another goroutine can evict the entry via `Set()` or `evictOldest()`, causing `MoveToFront` on a removed list element (undefined behavior in `container/list`). **Fix:** Use a single `Lock()` for the entire `Get()` operation.
- [ ] `ext_links.go:306-348` — **Unsafe links remain clickable**: Links with `StatusUnsafe` are still rendered as `<a href="...">` with the real URL. Users can click through to unsafe destinations. Images with `StatusUnsafe` are completely blocked. This inconsistency is a design concern. Consider rendering unsafe links as non-clickable or adding a confirmation interstitial.
- [ ] `status.go:151-152` — **No scan ID validation**: The `/api/safeurl/scan/{id}` endpoint extracts the scan ID from the URL path with no format/length validation. Arbitrary strings are passed to `validator.GetScanStatus()` and on to the SDK. An attacker could enumerate scan IDs to discover URLs being scanned by other users.
- [ ] `main.go:219-223` — **API key via CLI flag**: `--safeurl-api-key` passes the secret on the command line, visible in `ps aux` output. Use an environment variable (e.g., `GNOWEB_SAFEURL_API_KEY`) instead.
- [ ] `ext_safeurl.go:128-129` — **`context.Background()` in rendering pipeline**: The API call uses `context.Background()` instead of the HTTP request context, so request cancellation does not propagate to the SafeURL API call.
- [ ] `controller-safeurl.ts:47` — **No `encodeURIComponent` on scan ID**: The scan ID is interpolated directly into the fetch URL path. DOM manipulation could inject path traversal characters. Use `encodeURIComponent(scanId)`.
- [ ] Missing `client_test.go` — The `Client` struct has zero test coverage.

## Nits
- [ ] `client.go:62-73` — Error from SDK submit is logged as warning but swallowed; `error` in return signature is misleading since it effectively never returns one.
- [ ] `controller-safeurl.ts:57-59` — Recursive `setTimeout` polling continues even if the element is removed from the DOM. Check `element.isConnected` before polling.
- [ ] `ext.go:83` — `ExtSafeURL.Extend(m, validator)` has a non-standard signature that breaks the `goldmark.Extender` interface pattern (other extensions only take `m`).
- [ ] `cache.go` — No background eviction of expired entries. Expired entries accumulate until `maxSize` is hit or they are accessed.
- [ ] `validator.go` — No rate limiting on the number of URLs submitted per batch. A page with thousands of external links would submit all of them.

## Missing Tests
- [ ] `IsExternalURL` with bypass URLs (confirmed failing — see Critical #1) — `validator_test.go`
- [ ] Concurrency test for `Cache.Get()`/`Set()` with `-race` — `cache_test.go`
- [ ] `SetMulti()` — `cache_test.go`
- [ ] `Client` integration test with mock HTTP server — missing `client_test.go`
- [ ] `ValidateURLs()` with enabled validator and mock API — `validator_test.go`
- [ ] Markdown rendering output tests for each safety status (safe/unsafe/pending/unavailable) — missing `ext_safeurl_test.go`
- [ ] SafeURL polling endpoint tests — missing `status_test.go`

## Suggestions
- The `IsExternalURL` function should use `net/url.Parse` for proper URL parsing instead of string matching. This would fix the bypass and handle edge cases like userinfo, fragments, and query parameters — `validator.go:143-173`.
- Consider adding a `/api/safeurl/batch` endpoint that accepts multiple scan IDs to reduce polling overhead when a page has many pending links — `status.go`.
- The LRU cache could benefit from periodic background cleanup of expired entries to prevent memory waste — `cache.go`.
- The SafeURL feature adds a new third-party dependency (`safeurl-sdk`) to the main `go.mod` and `contribs/gnodev/go.mod`. Consider whether this should be a separate module or if the dependency footprint is acceptable for the monorepo.

## Questions for Author
- Is the decision to keep unsafe links clickable (with warning icon) intentional? If so, what's the rationale for the asymmetry with images (which are completely blocked)?
- Has the CSP issue been tested in strict mode? The `connect-src` policy appears to block the polling `fetch()` calls entirely.
- Is there a plan to support the SafeURL API key via environment variable instead of (or in addition to) the CLI flag?
- What happens when the SafeURL API rate-limits the gnoweb instance? Is there graceful degradation for quota exhaustion?

## Verdict
**REQUEST CHANGES** — The `IsExternalURL` substring-based domain check is a confirmed security bypass (6/6 adversarial test cases fail), the XSS in `controller-safeurl.ts`, and the CSP `connect-src` issue that breaks the feature in strict mode are all must-fix before merge.
