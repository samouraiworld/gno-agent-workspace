# Review: PR [#5861](https://github.com/gnolang/gno/pull/5861)
Event: APPROVE

## Body
Ran the store's concurrent add-and-flush path under `go test -race` on 0170d49a2; the mutex holds and the atomic write leaves no torn file.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5861-tm2-address-book/1-0170d49a2/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/p2p/discovery/store.go:110-113 [↗](../../../../../.worktrees/gno-review-5861/tm2/pkg/p2p/discovery/store.go#L110)
Re-adding a peer already in the store refreshes its `LastSeen` in memory but never marks the store dirty, so the update never reaches disk. Eviction ranks by `LastSeen` oldest-first, so after a restart an actively re-discovered peer with an old first-seen time is evicted ahead of a long-dead peer first seen more recently. Mark the store dirty whenever `LastSeen` changes, not only on first insert.

Missing test: eviction order survives a restart when a peer is re-discovered.

<details><summary>fix</summary>

Set `dirty` and bump `generation` unconditionally in `AddPeers`, so a `LastSeen` refresh persists:

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

`TestStore_ReAdd_PersistsLastSeen` flips red to green and the discovery and config suites stay clean under `-race`.
</details>

<details><summary>test cases</summary>

Fails red at 0170d49a2, passes once a `LastSeen` refresh marks the store dirty:

```go
func TestStore_ReAdd_PersistsLastSeen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "addrbook.json")

	s, err := NewStore(path, types.NetAddress{})
	require.NoError(t, err)

	addr := generateTestAddress(t, "1.2.3.4", 26656)

	s.AddPeers(addr)
	require.NoError(t, s.Save())
	first := persistedLastSeen(t, path, addr.String())

	// Advance past one-second Unix granularity, then re-discover the same peer.
	time.Sleep(1100 * time.Millisecond)
	s.AddPeers(addr)
	require.NoError(t, s.Save())
	second := persistedLastSeen(t, path, addr.String())

	require.Greater(t, second, first,
		"re-discovering a peer must refresh its persisted LastSeen, else eviction ranks it stale after restart")
}

func persistedLastSeen(t *testing.T, path, addr string) int64 {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var raw storeJSON
	require.NoError(t, json.Unmarshal(data, &raw))

	for _, e := range raw.Peers {
		if e.Addr == addr {
			return e.LastSeen
		}
	}

	t.Fatalf("address %s not found in persisted store", addr)
	return 0
}
```
</details>

## tm2/pkg/p2p/discovery/store.go:235-243 [↗](../../../../../.worktrees/gno-review-5861/tm2/pkg/p2p/discovery/store.go#L235)
A second corrupt-file load overwrites the `<file>.corrupt` backup written by the first, losing the earlier copy. A timestamp or counter suffix on the backup name keeps both.

<details><summary>fix</summary>

Timestamp the backup path so each corrupt load keeps its own copy:

```diff
-		corruptPath := s.filePath + ".corrupt"
+		// Timestamp the backup so a later corrupt load does not overwrite an
+		// earlier one.
+		corruptPath := fmt.Sprintf("%s.%d.corrupt", s.filePath, time.Now().UnixNano())
```

`TestStore_Load_CorruptFileTreatedAsEmpty` now globs `<file>.*.corrupt`; the suite passes under `-race`.
</details>

## tm2/pkg/p2p/config/config.go:17 [↗](../../../../../.worktrees/gno-review-5861/tm2/pkg/p2p/config/config.go#L17)
`defaultAddrBookPath` is only read, never reassigned, so a `const` fits and removes it as a mutable package global.

<details><summary>fix</summary>

```diff
-var defaultAddrBookPath = "config/addrbook.json"
+const defaultAddrBookPath = "config/addrbook.json"
```

`go build` and `go test ./tm2/pkg/p2p/config/...` both pass.
</details>
