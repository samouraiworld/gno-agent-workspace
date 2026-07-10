# Review: PR [#5928](https://github.com/gnolang/gno/pull/5928)
Event: APPROVE

## Body
Verified on cf7fb56bd: reverting the 1 ugnot fee back to zero, or the signature slice back to empty, makes each emitted tx fail ValidateBasic after the amino round-trip. Those are the two breaks this change fixes.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5928-fork-valoper-seed-payable/1-cf7fb56bd/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnogenesis/internal/fork/valoper_seed.go:367-372 [↗](../../../../../.worktrees/gno-review-5928/contribs/gnogenesis/internal/fork/valoper_seed.go#L367)
Missing test: no test round-trips an emitted line and calls ValidateBasic, which is the property this change adds. A revert to a zero fee or an empty signature slice would keep every test green while re-breaking replay.

<details><summary>test cases</summary>

Add to `contribs/gnogenesis/internal/fork/`; passes at cf7fb56bd, fails if either fix is reverted.

```go
package fork

import (
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/stretchr/testify/require"
)

func TestValoperSeed_EmittedTxPassesValidateBasic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	csv := validHeader + "\n" +
		opAddrA + "," + validPubKeyA + ",alice,Alice,cloud\n" +
		opAddrB + "," + validPubKeyB + ",bob,Bob,on-prem\n"

	out, err := runSeed(t, dir, csv)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	for i, line := range lines {
		var at AnnotatedTx
		require.NoError(t, amino.UnmarshalJSON([]byte(line), &at))
		require.NoErrorf(t, at.Tx.ValidateBasic(),
			"emitted valoper-seed tx %d must pass ValidateBasic", i)
	}
}
```
</details>

## contribs/gnogenesis/internal/fork/valoper_seed.go:150 [↗](../../../../../.worktrees/gno-review-5928/contribs/gnogenesis/internal/fork/valoper_seed.go#L150)
Suggestion: the help says the caller needs a balance of 1 ugnot, but every CSV row becomes its own tx and the ante handler [deducts that fee](https://github.com/gnolang/gno/blob/cf7fb56bd/tm2/pkg/sdk/auth/ante.go#L173-L184) from the same caller for each one. An N-row CSV needs the caller funded with N ugnot.
