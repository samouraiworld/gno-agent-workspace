# Review: PR #5839
Event: APPROVE

## Body
Looks good. Verified on 47f0596f0: reverting the `nodesManager.Delete(sid)` line leaks every finished node's store into the process-lifetime manager map, climbing the sequential live heap to ~9 GB / ~12.3 GB `Sys` and toward OOM while the suite still passes green; with the line, the map drains and the heap stays flat (~0.4 GB post-GC).

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5839-release-nodes-after-script/1-47f0596f0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)
