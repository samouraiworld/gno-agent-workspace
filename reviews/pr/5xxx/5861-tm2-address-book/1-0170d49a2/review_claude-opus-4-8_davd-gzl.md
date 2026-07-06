# PR [#5861](https://github.com/gnolang/gno/pull/5861): feat(tm2): implement address book

URL: https://github.com/gnolang/gno/pull/5861
Author: julienrbrt | Base: master | Files: 7 | +853 -9
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 0170d49a2 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5861 0170d49a2`

**TL;DR:** A gno.land node discovers peers over the network but forgets them on restart, so it has to rediscover from scratch every boot. This PR writes discovered peer addresses to a small JSON file (`config/addrbook.json`) and reloads and re-dials them on startup.

**Verdict: APPROVE** — solid, well-tested feature; one quality gap where a re-discovered peer's freshness timestamp never reaches disk, which decays the eviction order across restarts.

## Summary
Adds a `Store` (`tm2/pkg/p2p/discovery/store.go`) that persists discovered peer addresses to a JSON address book and reloads them so the node can re-dial known peers without re-running discovery. The discovery reactor now takes a store: it saves newly discovered peers on each tick, flushes on shutdown, and dials the persisted set on start. The store is concurrency-safe (`sync.RWMutex`), writes atomically (`WriteFileAtomic`, so a crash mid-write can't corrupt the file), filters the node's own address, caps at `defaultMaxPeers = 1000` with oldest-first eviction, and treats a corrupt file as empty after backing it up to `<file>.corrupt`. All the concerns raised by @tbruyelle on the PR (unbounded growth, the `time.Unix(0,0).IsZero()` dead condition, the `max` shadowing lint, silently overwriting a corrupt file, the dirty-flag race) are addressed at this head.

The one gap: `AddPeers` only marks the store dirty when a brand-new address key appears, so re-discovering a peer already in the store refreshes its `LastSeen` in memory but never writes it to disk. Eviction ranks by `LastSeen` (oldest first), so after a restart an actively-seen peer whose first-seen time is old can be evicted ahead of a long-dead peer that was first seen more recently, which is the opposite of what the feature wants.

## Glossary
- Address book — on-disk JSON list of discovered peer addresses, reloaded at boot.
- Discovery reactor — the p2p service that periodically asks peers for their peer lists (PEX).
- Eviction — dropping oldest entries once the store exceeds `maxPeers`.

## Critical (must fix)
None.

## Warnings (should fix)
- **[re-discovered peers look stale after a restart]** `store.go:110-113` — Re-adding an existing peer refreshes `LastSeen` in memory but is never persisted, so eviction ranks a long-lived peer as stale after restart.
  <details><summary>details</summary>

  `AddPeers` sets `s.dirty = true` and bumps `generation` only inside the `if _, exists := s.peers[key]; !exists` branch. Updating an existing peer's `LastSeen` at [`store.go:115`](https://github.com/gnolang/gno/blob/0170d49a2/tm2/pkg/p2p/discovery/store.go#L115) · [↗](../../../../../.worktrees/gno-review-5861/tm2/pkg/p2p/discovery/store.go#L115) leaves `dirty` unchanged, so the next `Save` is a no-op and the persisted timestamp stays frozen at first-seen. Eviction sorts oldest-`LastSeen` first at [`store.go:207-209`](https://github.com/gnolang/gno/blob/0170d49a2/tm2/pkg/p2p/discovery/store.go#L207-L209) · [↗](../../../../../.worktrees/gno-review-5861/tm2/pkg/p2p/discovery/store.go#L207-L209), and `load` restores the frozen timestamps, so across restarts an actively re-discovered peer with an old first-seen time is evicted before a dead peer first seen more recently. This undermines the feature's goal of keeping good peers. Confirmed with a red test: `store_lastseen_persist_test.go`. Fix: mark the store dirty (and bump generation) whenever `LastSeen` changes, not only on first insert.

  Fix applied in the review worktree:

  ```diff
  		key := addr.String()
  -		if _, exists := s.peers[key]; !exists {
  -			s.dirty = true
  -			s.generation++
  -		}
  +		s.dirty = true
  +		s.generation++

  		s.peers[key] = &knownAddress{Addr: addr, LastSeen: time.Now()}
  ```

  Verified: the ported `TestStore_ReAdd_PersistsLastSeen` flips from red to green (persisted `LastSeen` advances from `1783324794` to a later value across the re-add), and `go test -race ./tm2/pkg/p2p/discovery/... ./tm2/pkg/p2p/config/...` stays clean.
  </details>

## Nits
- `store.go:207-209` — Eviction uses `sort.Slice` (not stable) and `LastSeen` is persisted at one-second Unix granularity, so among peers sharing a second the dropped ones are arbitrary. Off-chain path, no consensus impact; only affects which of several same-second peers survive.
- `store.go:235-243` — A second corrupt-file load overwrites the `<file>.corrupt` backup from the first, losing the earlier copy. Rare, but a suffix or timestamp on the backup name would preserve both. Fixed in the worktree: the backup path is now `fmt.Sprintf("%s.%d.corrupt", s.filePath, time.Now().UnixNano())`, so each corrupt load keeps its own copy. The PR's own `TestStore_Load_CorruptFileTreatedAsEmpty` was updated to glob `<file>.*.corrupt` for the backup; the suite passes under `-race`.

## Missing Tests
- **[eviction order survives a restart]** `store.go:110-113` — No test asserts that a re-discovered peer's refreshed `LastSeen` is persisted, which is exactly the gap in the Warning above.
  <details><summary>details</summary>

  The existing suite covers save/reload, eviction by count, and idempotent re-add, but never checks that re-adding refreshes the persisted timestamp. The ready-to-add test [`store_lastseen_persist_test.go`](../../../../../reviews/pr/5xxx/5861-tm2-address-book/1-0170d49a2/tests/store_lastseen_persist_test.go) fails red at 0170d49a2 and passes once `LastSeen` refreshes mark the store dirty. It re-adds one peer after a one-second sleep and asserts the persisted `LastSeen` advanced.
  </details>

## Suggestions
- `config/config.go:17` — `defaultAddrBookPath` is a package-level `var` but is only read, never reassigned. A `const` documents the intent and removes it as a mutable global.
  <details><summary>details</summary>

  It is read at [`config.go:76`](https://github.com/gnolang/gno/blob/0170d49a2/tm2/pkg/p2p/config/config.go#L76) · [↗](../../../../../.worktrees/gno-review-5861/tm2/pkg/p2p/config/config.go#L76) and [`config.go:108`](https://github.com/gnolang/gno/blob/0170d49a2/tm2/pkg/p2p/config/config.go#L108) · [↗](../../../../../.worktrees/gno-review-5861/tm2/pkg/p2p/config/config.go#L108) and never written. A string `const` is the natural fit. Applied in the worktree: `var defaultAddrBookPath = ...` becomes `const defaultAddrBookPath = ...`; `go build` and `go test ./tm2/pkg/p2p/config/...` both pass.
  </details>

## Open questions
- Eviction runs only on `AddPeers`; a store loaded from a hand-edited or old file that already exceeds `maxPeers` is not trimmed until the next add. Benign (the next discovery tick adds a peer and trims), not posted.

## Invariant catalog

Tm2 Go p2p networking PR, off-chain, no gno-code path. Walked the catalog: gas, realm state safety, caller/access control, coin/banker, storage deposit, VM-fault recoverability, VM-semantics-vs-Go, and type-check/preprocess classes do not apply (no VM, stdlib, consensus, or `.gno` surface). Relevant classes:

- Determinism: the persisted address book is off-chain and never enters a consensus or output path, so the `time.Now()` timestamps, map-iteration order in `GetPeers`/`Save`, and non-stable `sort.Slice` in `evict` carry no consensus-determinism risk. The map-iteration and sort-tie nondeterminism only affect which peers a single node keeps, noted as a Nit.
- Global mutable state & concurrency: `Store` is built for concurrent use behind a `sync.RWMutex`; `go test -race` on the discovery and config packages is clean (see below). `WriteFileAtomic` uses randomized temp names plus rename, so concurrent `Save`/`Flush` calls cannot corrupt the file (last rename wins). `defaultAddrBookPath` is a package `var` but read-only after init, safe; flagged as a Suggestion to make it `const`.
- Error & panic handling: `.go` error discipline holds. `load` returns real errors for stat/read failures and degrades gracefully on a corrupt JSON file (logs, backs up, treats as empty) rather than panicking; the reactor logs and continues on `Save` failures. No swallowed errors.

## Verification

Verified on 0170d49a2:
- `go test -race -count=1 ./tm2/pkg/p2p/discovery/... ./tm2/pkg/p2p/config/...` passes; the store's concurrent add+flush test exercises the mutex under `-race`.
- The LastSeen-not-persisted Warning reproduces: `store_lastseen_persist_test.go` fails red at this head (persisted `LastSeen` unchanged after a re-add one second later).
- Self-address filter is wired correctly: `makeNodeInfo` populates `nodeInfo.NetAddress` with the node's real ID before `NewStore` receives it, so `NetAddress.Same` (dial-string or ID match) filters the node's own address on both add and load.

All three actionable findings (the LastSeen-persist Warning, the corrupt-backup Nit, the `const` Suggestion) were fixed in the review worktree and re-verified: `go test -race -count=1 ./tm2/pkg/p2p/discovery/... ./tm2/pkg/p2p/config/...` passes, and the ported `TestStore_ReAdd_PersistsLastSeen` now goes green.
