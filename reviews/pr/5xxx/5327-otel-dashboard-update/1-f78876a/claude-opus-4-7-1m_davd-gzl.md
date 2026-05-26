# PR #5327: chore(telemetry): Updating reference OTEL Dashboard

URL: https://github.com/gnolang/gno/pull/5327
Author: sw360cab | Base: master | Files: 4 | +41 -124
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]

**Verdict: REQUEST CHANGES** — the `state.go` "fix" is a no-op (still truncates to `int64` ms, so sub-ms builds keep recording 0); dashboard cleanup and image pinning are sound but ship with a few mislabeled panels.

## Summary

Telemetry reference stack maintenance: pin all four `docker-compose` images off `latest` (collector `0.147.0`, tempo `2.9.0`, prometheus `v3.9.1`, grafana `10.4.2`, supernova `1.5.3`), tweak the `gnoland-val` healthcheck timing (`60s`→`90s` start, `10s`→`20s` interval) and add `--log-level=info` + `--depth 1` to the genesis-bootstrap script, drop two dashboard panels that target metrics not emitted by the node (`dialing_peers_gauge`, `broadcast_tx_hist_milliseconds_*`), replace the broadcast panel with an "Average Gas Price" panel querying `block_gas_price_hist_token_*`, and add a sort + pagination to the traces panel.

The one Go-code change — `tm2/pkg/bft/consensus/state.go:1010-1012` — claims to record fractional milliseconds for `BuildBlockTimer` but the receiver is `metric.Int64Histogram`, so the final `int64(durationMs)` cast truncates back to whole milliseconds. Old code (`time.Since(t).Milliseconds()`) and new code emit byte-identical values for every input; the "always 0" symptom called out in the PR description is unchanged. Either swap to `Float64Histogram` or change the metric to microseconds (`WithUnit("us")` + `time.Since(t).Microseconds()`).

## Glossary

- `BuildBlockTimer` — OTEL `Int64Histogram` registered with `WithUnit("ms")` in [`tm2/pkg/telemetry/metrics/metrics.go:151-157`](../../../../../.worktrees/gno-review-5327/tm2/pkg/telemetry/metrics/metrics.go#L151-L157); the value is the block-build duration as recorded from `createProposalBlock`.
- `block_gas_price_hist_token` — Prometheus exposition of the `gasPriceKey` histogram (`WithUnit("token")` → suffix `_token`); the recorded value is the gas amount `gp.Gas` not a coin amount, see [`tm2/pkg/sdk/auth/keeper.go:335-349`](../../../../../.worktrees/gno-review-5327/tm2/pkg/sdk/auth/keeper.go#L335-L349).
- `traceql` — Tempo's query language; `{duration>80ms && resource.service.name=~".+"}` matches every span longer than 80ms, regardless of whether it's an RPC handler or a consensus span.

## Fix

Before: `metrics.BuildBlockTimer.Record(ctx, time.Since(t).Milliseconds())` — Go's `Duration.Milliseconds()` is integer-truncating, so any sub-ms build records 0. After: the code computes `durationMs := float64(d.Microseconds()) / 1000.0` and records `int64(durationMs)`. Because the histogram type is fixed as `Int64Histogram`, the `int64(...)` cast applies the same floor truncation as the original `.Milliseconds()`, so every input produces an identical observation. The dashboard side removes two panels whose metrics never existed in source (`dialing_peers_gauge`, `broadcast_tx_hist_milliseconds_*`) and pins images, both genuinely useful.

## Benchmarks / Numbers

Empirical comparison of old vs new on representative sub-ms inputs (`go run` of the two snippets in isolation):

| duration | old `Milliseconds()` | new `int64(µs/1000.0)` | fractional `µs/1000.0` |
|---|---|---|---|
| 400µs | 0 | 0 | 0.4 |
| 500µs | 0 | 0 | 0.5 |
| 999µs | 0 | 0 | 0.999 |
| 1.5ms | 1 | 1 | 1.5 |
| 1.999ms | 1 | 1 | 1.999 |
| 2.5ms | 2 | 2 | 2.5 |

## Critical (must fix)

- **[fix is a no-op]** [`tm2/pkg/bft/consensus/state.go:1010-1012`](../../../../../.worktrees/gno-review-5327/tm2/pkg/bft/consensus/state.go#L1010-L1012) — converting µs to float ms then casting back to `int64` truncates identically to `.Milliseconds()`; sub-ms builds still record 0.
  <details><summary>details</summary>

  The PR description states the goal is "Fixing issue with one metrics which produces always 0". The diff replaces `time.Since(t).Milliseconds()` with `int64(float64(time.Since(t).Microseconds()) / 1000.0)`. Because `BuildBlockTimer` is declared as `metric.Int64Histogram` at [`tm2/pkg/telemetry/metrics/metrics.go:73`](../../../../../.worktrees/gno-review-5327/tm2/pkg/telemetry/metrics/metrics.go#L73) and initialized via `meter.Int64Histogram(...)` at [`tm2/pkg/telemetry/metrics/metrics.go:151-157`](../../../../../.worktrees/gno-review-5327/tm2/pkg/telemetry/metrics/metrics.go#L151-L157), the final `int64(durationMs)` cast applies a floor that produces the same value as the original truncation in every case (table above). The `// Record as fractional milliseconds - otherwise the result will be truncated` comment is misleading — fractional milliseconds are *lost* at the cast, not preserved. On a fast node where `createProposalBlock` consistently runs under 1ms (the most plausible cause of the "always 0" symptom this PR aims to fix), the metric continues to emit 0 after this change.

  **Repro:** from a local clone of gnolang/gno:
  ```bash
  gh pr checkout 5327 -R gnolang/gno
  cat > /tmp/buildblock_truncation_test.go <<'EOF'
  package main

  import (
  	"fmt"
  	"time"
  )

  func main() {
  	for _, d := range []time.Duration{
  		400 * time.Microsecond,
  		500 * time.Microsecond,
  		999 * time.Microsecond,
  		1500 * time.Microsecond,
  	} {
  		oldVal := d.Milliseconds()
  		durationMs := float64(d.Microseconds()) / 1000.0
  		newVal := int64(durationMs)
  		fmt.Printf("d=%v  old=%d  new=%d  parity=%v\n", d, oldVal, newVal, oldVal == newVal)
  	}
  }
  EOF
  go run /tmp/buildblock_truncation_test.go
  rm /tmp/buildblock_truncation_test.go
  ```
  Expected output: every row reports `parity=true`, confirming the rewrite emits the same `int64` as the original.

  Fix: choose one of the two correct paths and align the metric registration with the recorded unit.
  1. Sub-ms accuracy as float ms — switch the field to `metric.Float64Histogram` and use `meter.Float64Histogram(...)`; then record `float64(time.Since(t).Microseconds()) / 1000.0` directly (drop the int cast).
  2. Sub-ms accuracy as integer µs — keep `Int64Histogram` but change the unit to `metric.WithUnit("us")` at [`tm2/pkg/telemetry/metrics/metrics.go:151-157`](../../../../../.worktrees/gno-review-5327/tm2/pkg/telemetry/metrics/metrics.go#L151-L157) and record `time.Since(t).Microseconds()`. The Prometheus series will be renamed (`build_block_hist_us_*`), so any existing query/alert referring to the old name must be updated in the dashboard JSON too.
  </details>

## Warnings (should fix)

- **[panel mislabeled — averages gas amounts, not gas price]** [`misc/telemetry/grafana/provisioning/dashboards/gno-otel-dashboards.json:992-1041`](../../../../../.worktrees/gno-review-5327/misc/telemetry/grafana/provisioning/dashboards/gno-otel-dashboards.json#L992-L1041) — the new "Average Gas Price" panel uses unit `gnot` but queries `block_gas_price_hist_token_*`, whose recorded value is `gp.Gas` (gas amount) not a coin amount.
  <details><summary>details</summary>

  At [`tm2/pkg/sdk/auth/keeper.go:335-349`](../../../../../.worktrees/gno-review-5327/tm2/pkg/sdk/auth/keeper.go#L335-L349) the histogram is recorded with `gp.Gas` (an int64 unit count); the actual coin price is shipped as the `Coin` attribute (`gp.Price.String()`). So `rate(_sum)/rate(_count)` produces an average gas amount, not a price. On top of that, the panel unit is set to `gnot` while gas prices in tm2/gno.land are denominated in `ugnot` (10⁶ ugnot = 1 gnot); the threshold value of 1000 makes more sense as an alert on a count of gas units than as a price boundary in gnot. The metric registration at [`tm2/pkg/telemetry/metrics/metrics.go:254-260`](../../../../../.worktrees/gno-review-5327/tm2/pkg/telemetry/metrics/metrics.go#L254-L260) uses `WithUnit("token")`, which is itself ambiguous (token of what?). Fix: rename the panel to "Average Block Gas Amount", set the unit to `short` (or remove it), and either drop the threshold or recalibrate it against expected per-block gas. If a coin-denominated gas price panel is genuinely wanted, it has to be derived from the `Coin` attribute string, which is non-trivial in PromQL — easier to add a dedicated histogram in `logTelemetry` that records `gp.Price.Amount` in ugnot.
  </details>

- **[traces panel title doesn't match the query]** [`misc/telemetry/grafana/provisioning/dashboards/gno-otel-traces-dashboards.json:99-106`](../../../../../.worktrees/gno-review-5327/misc/telemetry/grafana/provisioning/dashboards/gno-otel-traces-dashboards.json#L99-L106) — title "RPC Requests Traces (> 80 ms)" but the TraceQL filter is `{duration>80ms && resource.service.name=~".+"}`, which matches every span longer than 80ms regardless of whether it originated in the RPC layer.
  <details><summary>details</summary>

  The query has no `name=~"RPC.*"` or service/span filter narrowing to JSON-RPC handlers; `resource.service.name=~".+"` accepts every service. In practice the table will surface consensus, mempool, and ABCI spans alongside RPC ones. Two acceptable fixes: (a) tighten the query to match RPC spans only (e.g. `{span.name=~"jsonrpc.*" && duration>80ms}`, depending on actual span naming convention emitted by the node), or (b) rename the panel to "Slow Spans (> 80ms)" to match the query. The previous `${height}` template variable was correctly removed; this is a different drift introduced in the same edit.
  </details>

## Nits

- [`misc/telemetry/grafana/provisioning/dashboards/gno-otel-dashboards.json:990`](../../../../../.worktrees/gno-review-5327/misc/telemetry/grafana/provisioning/dashboards/gno-otel-dashboards.json#L990) — `"unit": "gnot"` is not a Grafana built-in unit; will render as plain text suffix instead of triggering Grafana's coin/decimal formatting. Pair with the gas-amount fix above.
- [`misc/telemetry/docker-compose.yml:38`](../../../../../.worktrees/gno-review-5327/misc/telemetry/docker-compose.yml#L38) — `grafana/grafana:10.4.2` is the only image pinned to a non-latest minor that's already old (10.x EOL'd before 11.x); other images here are recent. If pinning to be reproducible, consider `11.x` to line up with the dashboard `pluginVersion: 11.2.0` / `11.5.2` in the JSON.
- [`misc/telemetry/docker-compose.yml:65`](../../../../../.worktrees/gno-review-5327/misc/telemetry/docker-compose.yml#L65) — adding `--depth 1` is a nice speedup, but the surrounding script still `make`s `gnogenesis` from the cloned source. If the goal is faster bring-up, the longer-term move is to use the existing `ghcr.io/gnolang/gno/gnoland:master` image (already pulled for the node itself) and skip the source clone entirely.
- [`misc/telemetry/grafana/provisioning/dashboards/gno-otel-traces-dashboards.json:99`](../../../../../.worktrees/gno-review-5327/misc/telemetry/grafana/provisioning/dashboards/gno-otel-traces-dashboards.json#L99) — the `limit: 20` field plus `spss: 40` (spans-per-spanset) interact in a way that's worth a one-line panel description; reviewers reading the JSON later will not remember which one bounds what.

## Missing Tests

- None. Telemetry/dashboard changes don't have a test surface in this repo, and the consensus path is exercised by existing `tm2/pkg/bft/consensus` tests (build/run confirmed green locally). The "fix is a no-op" critical above is best caught by a one-line unit assertion in a new `state_telemetry_test.go` that records a 500µs duration through `BuildBlockTimer` and asserts a non-zero observation — but writing it requires the histogram change first.

## Suggestions

- [`tm2/pkg/telemetry/metrics/metrics.go:151-157`](../../../../../.worktrees/gno-review-5327/tm2/pkg/telemetry/metrics/metrics.go#L151-L157) — when fixing `BuildBlockTimer`, audit the other `Int64Histogram` instruments that record millisecond durations (`HTTPRequestTime`, `WSRequestTime`) for the same truncation risk on fast paths. If gno.land's RPC handlers commonly finish in sub-ms, the same dashboard panels will render 0.
  <details><summary>details</summary>

  Both `httpRequestTimeKey` and `wsRequestTimeKey` are registered with `metric.WithUnit("ms")` and almost certainly populated with `.Milliseconds()` at the call site. The fix mechanism (switch to `Float64Histogram` or convert to µs) generalizes; doing all three in one PR keeps the dashboard panels honest.
  </details>

## Questions for Author

- @sw360cab — did you confirm the "always 0" symptom is gone on a node running this patch? On a fast machine `createProposalBlock` should consistently finish in sub-ms and, per the table above, the recorded value should still be 0. If your repro shows non-zero values now, what's different about the test environment?
- The PR description mentions "removing non-existing metrics and adding new ones". Beyond the gas-price panel, are there any planned additions (gnovm gas, mempool depth, validator vote latency) that this dashboard should pick up while it's being edited?
