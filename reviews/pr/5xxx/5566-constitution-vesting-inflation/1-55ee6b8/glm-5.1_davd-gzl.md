# PR #5566: docs(constitution): $GNOT vesting schedule and inflation

**URL:** https://github.com/gnolang/gno/pull/5566
**Author:** dongwon8247 | **Base:** master | **Files:** 1 | **+12 -6**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR modifies `docs/CONSTITUTION.md` with three substantive changes:

1. **Vesting schedule**: Adds a common vesting schedule for all Genesis $GNOT allocations — 7% on the day $GNOT becomes transferrable, 7% each subsequent month, and 9% in the final month (fully vested 13 months after mainnet). Also clarifies that whitelisted funds remain subject to the vesting schedule.

2. **Inflation start date**: Shifts the start of $GNOT deflationary inflation from "the date of launch" to "one year after $GNOT becomes transferrable (aka the mainnet)". The variable Y is redefined as "year from inflation start" rather than "year from launch".

3. **Cumulative inflation labels**: Relabels "After N years" to "After N years of inflation" to remove ambiguity about the reference point, now that inflation start and launch are different dates.

The PR is documentation-only; no code or .gno files are affected.

## Test Results

- **Existing tests:** PASS (CI all green — build, check, docs, e2e-test)
- **Edge-case tests:** Skipped (docs-only PR)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `docs/CONSTITUTION.md:131-133` — The vesting schedule description is ambiguous about how many months pay 7%. "7% each subsequent month, and 9% in the final month" could mean (a) months 1–12 at 7%, month 13 at 9% (7×13 + 9 = 100%) or (b) months 1–11 at 7%, month 12 at 9% (7×12 + 9 = 93%). Interpretation (a) sums to 100% and matches "13 months after the mainnet", but the text does not make this explicit. Consider rewording to something like: "7% at month 0, 7% for each of the next 12 months (months 1–12), and 9% at month 13" to eliminate ambiguity.

## Nits

- [ ] `docs/CONSTITUTION.md:128` — Minor grammar: "the operation of the chain" was already correct; the original text had "the operation of the chain" as well (line 128 on master reads the same). The diff only adds a new sentence after it, so no issue — but note the new text appends "Whitelisted funds remain subject to the vesting schedule below." on the same line, creating a long line that could benefit from a line break before "Whitelisted".

## Missing Tests

None (docs-only PR)

## Suggestions

- Consider adding a vesting schedule table or timeline to make the schedule immediately clear without arithmetic. For example:

  | Time | % Vested | Cumulative |
  |------|----------|------------|
  | Mainnet (month 0) | 7% | 7% |
  | Month 1 | 7% | 14% |
  | ... | 7% | ... |
  | Month 12 | 7% | 91% |
  | Month 13 | 9% | 100% |

- The inflation section now uses "$GNOT becomes transferrable (aka the mainnet)" as the anchor date, which is consistent with the vesting section. However, other parts of the Constitution still use "launch" (e.g., lines 802–806 in the Oversight Body section: "Within 2 years after launch", "If after 2 years after launch"). Consider whether these references should also be clarified to mean "after mainnet" for consistency, or note that "launch" and "mainnet" are distinct events.

## Questions for Author

- Does the vesting schedule apply equally to the airdrop allocations (Airdrop1: 35%, Airdrop2: 23.1%)? The text says "All Genesis $GNOT allocations" which would include them, but airdrop recipients may expect immediate access. Clarifying this explicitly would be helpful.
- Is there a reason the final month is 9% rather than 7%, creating a non-uniform schedule? If the intent is 12×7% + 1×9% + 1×7% = 100%, a uniform 7.69% monthly schedule might be simpler (though less clean). If it's 13×7% + 9% = 100%, consider stating the month count directly.

## Verdict

APPROVE — The changes are internally consistent, the inflation math checks out (90.32M, 217.09M, 333.29M cumulative figures verified), and the clarifications improve the Constitution. The vesting schedule wording has a minor ambiguity that should be resolved before merge, but it's not blocking.
