# Review: [#6135](https://github.com/gnolang/gno/pull/6135)
Event: APPROVE

## Body
The path filter on [`ci-gnoe2e.yml`](https://github.com/gnolang/gno/blob/ddc5acfb9/.github/workflows/ci-gnoe2e.yml#L12-L29) matches a file in 84 of the 100 pull requests merged between 8 July and 4 September, so a validator that fails to boot reddens branches with no gnoe2e change.

<details><summary>checks that held</summary>

- The unit lane is green under the race detector, `go test -race -short ./internal/... .` from `misc/gnoe2e`, six packages ok. The workflow passes `-p 1 -timeout 25m -coverpkg=...` and no `-race`, so nothing in it reports that.
- `go run . defaults` walks `config.Config` and the auth, vm and bank genesis params by reflection and prints every settable path with its value. It resolves clean, names the three harness-assigned keys as unsettable, and reports `config.rpc.timeout_broadcast_tx_commit: 10s`, the constant [`patient_oracle.txtar:38-39`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/testdata/oracle/patient_oracle.txtar#L38-L39) rests its `request timeout` assertion on.
- `ts.Fatalf` panics with a package sentinel rather than calling `FailNow`, at [`testscript.go:1213`](https://github.com/rogpeppe/go-internal/blob/v1.15.0/testscript/testscript.go#L1213), so the `recover` in [`runCmdIteration`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/integration/commands.go#L119-L128) catches a failed attempt and `eventually` retries rather than ending the script.
</details>

## misc/gnoe2e/internal/termlog/handler.go:167-191 [gh](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/termlog/handler.go#L167-L191) · [↗](../../../../../.worktrees/gno-review-6135/misc/gnoe2e/internal/termlog/handler.go#L167-L191)
Nit: both derived handlers rebuild the struct field by field and omit `color`, which [`newHandler`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/termlog/handler.go#L57) sets, so [`write`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/termlog/handler.go#L81-L86) strips the escapes out of every line from the cluster loggers at [`cmd_run.go:456`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/cmd_run.go#L456) and [`cmd_serve.go:70`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/cmd_serve.go#L70), and on a terminal `make scenarios` colours the run's own lines and not the cluster's.

```suggestion
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		w:       h.w,
		level:   h.level,
		attrs:   append(slices.Clone(h.attrs), attrs...),
		mu:      h.mu,
		verbose: h.verbose,
		color:   h.color,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	// slog requires an empty name to be a no-op. Left to the branch below it
	// would label every later record with a group that was never opened.
	if name == "" {
		return h
	}
	// Not used in our codebase; treat as attr prefix.
	return &Handler{
		w:       h.w,
		level:   h.level,
		attrs:   append(slices.Clone(h.attrs), slog.String("group", name)),
		mu:      h.mu,
		verbose: h.verbose,
		color:   h.color,
	}
}
```

<details><summary>repro</summary>

[`TestHandlerColoursALineForATerminal`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/termlog/handler_test.go#L109) calls `newHandler` and never derives, so nothing in the package reaches this. The test that does:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6135 -R gnolang/gno
cat > misc/gnoe2e/internal/termlog/handler_derived_test.go <<'GO'
package termlog

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

// The baseline and the defect in one run: the undecorated logger is the
// existing TestHandlerColoursALineForATerminal, and the derived one is what
// every cluster and serve line actually goes through.
func TestHandlerKeepsColourThroughWith(t *testing.T) {
	var out bytes.Buffer
	// newHandler with color=true stands in for a run attached to a terminal,
	// which a bytes.Buffer can never be.
	root := slog.New(newHandler(&out, false, true))

	root.Info("cluster ready")
	require.Contains(t, out.String(), colorGreen+"INF"+colorReset,
		"baseline: the undecorated logger colours its level")

	out.Reset()
	root.With("component", "cluster").Info("cluster ready")
	require.Contains(t, out.String(), colorGreen+"INF"+colorReset,
		"a logger derived with With() must colour its level too")
	require.Contains(t, out.String(), colorBold+"cluster"+colorReset,
		"and must render the component prefix in bold")
}

func TestHandlerKeepsColourThroughGroup(t *testing.T) {
	var out bytes.Buffer
	slog.New(newHandler(&out, false, true)).WithGroup("cluster").Info("cluster ready")

	require.Contains(t, out.String(), colorGreen+"INF"+colorReset,
		"a logger derived with WithGroup() must colour its level too")
}
GO
go test -C misc/gnoe2e -count=1 -run 'TestHandlerKeepsColour' ./internal/termlog/
rm misc/gnoe2e/internal/termlog/handler_derived_test.go
```

The undecorated baseline holds and both derived handlers fail:

```
--- FAIL: TestHandlerKeepsColourThroughWith (0.00s)
    handler_derived_test.go:24:
        	Error:      	"INF cluster: cluster ready\n" does not contain "\x1b[32mINF\x1b[0m"
        	Messages:   	a logger derived with With() must colour its level too
--- FAIL: TestHandlerKeepsColourThroughGroup (0.00s)
    handler_derived_test.go:34:
        	Error:      	"INF cluster ready\n" does not contain "\x1b[32mINF\x1b[0m"
        	Messages:   	a logger derived with WithGroup() must colour its level too
FAIL	github.com/gnolang/gno/misc/gnoe2e/internal/termlog	0.012s
```
</details>

## misc/gnoe2e/internal/cluster/cluster.go:169-174 [gh](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/cluster/cluster.go#L169-L174) · [↗](../../../../../.worktrees/gno-review-6135/misc/gnoe2e/internal/cluster/cluster.go#L169-L174)
Nit: `NumValidators` is bounded below at one and not above, so the [ceiling of sixteen](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/integration/clusterspec.go#L30) that [`parseClusterSection`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/integration/clusterspec.go#L206-L211) puts on a declared count never reaches the `-validators` flag, and `run -validators 17` validates clean into [`StartCluster`'s setup loop](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/internal/cluster/cluster.go#L382-L388).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6135 -R gnolang/gno
cat > misc/gnoe2e/internal/integration/override_bound_test.go <<'GO'
package integration

import (
	"testing"

	"github.com/gnolang/gno/misc/gnoe2e/internal/cluster"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/stretchr/testify/require"
)

const overTheCap = maxValidators + 1

// The script route and the flag route reach the same cfg.NumValidators, so the
// bound belongs to whichever of them a caller used, not to the section alone.
func TestValidatorsOverrideKeepsTheSectionsBound(t *testing.T) {
	script := []byte("-- cluster --\nvalidators: 17\n")
	_, err := ParseClusterSpec(script)
	require.ErrorContains(t, err, "at most 16",
		"baseline: the section refuses a count past the cap")

	n := overTheCap
	spec := ClusterOverrides{Validators: &n}.Apply(ClusterSpec{Validators: 4})

	cfg := cluster.DefaultClusterConfig()
	require.NoError(t, spec.ApplyTo(&cfg, crypto.Address{}, crypto.Address{}))
	require.Equal(t, overTheCap, cfg.NumValidators)

	require.Error(t, cfg.Validate(),
		"-validators past the cap must be refused the way the section refuses it")
}
GO
go test -C misc/gnoe2e -count=1 -run TestValidatorsOverrideKeepsTheSectionsBound ./internal/integration/
rm misc/gnoe2e/internal/integration/override_bound_test.go
```

The section refuses seventeen and the flag route validates clean:

```
--- FAIL: TestValidatorsOverrideKeepsTheSectionsBound (0.00s)
    override_bound_test.go:28:
        	Error:      	An error is expected but got nil.
        	Test:       	TestValidatorsOverrideKeepsTheSectionsBound
        	Messages:   	-validators past the cap must be refused the way the section refuses it
FAIL	github.com/gnolang/gno/misc/gnoe2e/internal/integration	0.059s
```

`maxValidators` lives in `internal/integration`, which imports `internal/cluster`, so the constant moves down before `Validate` can read it.
</details>

## misc/gnoe2e/docs/writing-scenarios.md:69 [gh](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/docs/writing-scenarios.md?plain=1#L69) · [↗](../../../../../.worktrees/gno-review-6135/misc/gnoe2e/docs/writing-scenarios.md#L69)
Nit: only the `vm/qfile` path prints `package %q is not available`, at [`keeper.go:1812`](https://github.com/gnolang/gno/blob/ddc5acfb9/gno.land/pkg/sdk/vm/keeper.go#L1812), and a `vm/qrender` of the same parked package answers `package not found` at [`keeper.go:1751`](https://github.com/gnolang/gno/blob/ddc5acfb9/gno.land/pkg/sdk/vm/keeper.go#L1751), which is the form [`tour.txtar:167-168`](https://github.com/gnolang/gno/blob/ddc5acfb9/misc/gnoe2e/testdata/tour.txtar#L167-L168) pins.

```suggestion
A `vm/qfile` read of a parked or missing package prints `package "..." is not available` on stdout; pin that:
```
