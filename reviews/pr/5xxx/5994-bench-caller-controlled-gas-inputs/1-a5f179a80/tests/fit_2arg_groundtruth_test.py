# Run: from a gno checkout:
# gh pr checkout 5994 -R gnolang/gno && git checkout a5f179a80
# curl -fsSL -o gnovm/cmd/calibrate/fit_2arg_groundtruth_test.py \
#   https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5994-bench-caller-controlled-gas-inputs/1-a5f179a80/tests/fit_2arg_groundtruth_test.py
# python3 gnovm/cmd/calibrate/fit_2arg_groundtruth_test.py
# rm gnovm/cmd/calibrate/fit_2arg_groundtruth_test.py
#
# Feeds NATIVE_SPECS_2ARG noise-free data generated from each native's real
# cost model, so the assertions describe the fitter and not the bench machine.
# innerHash is one sha256 over the concatenation, so an additive plane is the
# true model and the recovered slopes must be equal and correct. modExp costs
# one modular squaring per exponent bit, each quadratic in the modulus, which
# no additive plane can bound. At a5f179a80 the second assertion fails: the
# emitted row undercharges its own worst sampled point and is emitted anyway.

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import gen_native_table as G  # noqa: E402

INNER_GRID = [(32, 32), (256, 256), (1024, 1024), (4096, 4096),
              (32, 4096), (4096, 32), (32, 1024), (1024, 32)]
MODEXP_GRID = [(32, 32), (64, 64), (128, 128), (256, 256), (512, 512),
               (32, 256), (1024, 256), (256, 32), (256, 512)]

# Per-byte and base taken from the shipped crypto/merkle.leafHash row, which
# hashes the same way over one argument instead of two.
INNER_BASE, INNER_PER_BYTE = 3528.0, 32.14
# Scaled so a 256-byte modulus with a 256-byte exponent costs 6.16ms, the
# figure the shipped modExp row's own comment cites.
MODEXP_BASE = 58000.0
MODEXP_C = 6.16e6 / (256 * 256 * 256)


def inner_truth(left, right):
    return INNER_BASE + INNER_PER_BYTE * (1 + left + right)


def modexp_truth(exp, mod):
    return MODEXP_BASE + MODEXP_C * exp * mod * mod


def grid_of(pairs, truth):
    return {(n1, n2): [truth(n1, n2)] for n1, n2 in pairs}


def charged(base, s1_per_1024, s2_per_1024, n1, n2):
    """Replays the runtime's integer arithmetic in chargeNativeGas."""
    return base + s1_per_1024 * n1 // 1024 + s2_per_1024 * n2 // 1024


failures = []


def check(name, ok, detail):
    print(("PASS" if ok else "FAIL") + f" {name}: {detail}")
    if not ok:
        failures.append(name)


# --- innerHash: an additive plane is the true model, so the fit must be exact.
base, s1, s2, r2, separable = G.fit_2arg(grid_of(INNER_GRID, inner_truth))
s1_per_1024, s2_per_1024 = round(s1 * 1024), round(s2 * 1024)
check("innerHash grid is separable", separable,
      "off-diagonal pairs make the two design columns independent")
check("innerHash slopes are symmetric", abs(s1 - s2) < 0.01 * s1,
      f"s1={s1:.4f} s2={s2:.4f} ns/byte")
check("innerHash slopes match the hash rate",
      abs(s1 - INNER_PER_BYTE) < 0.01 * INNER_PER_BYTE,
      f"fitted {s1:.4f} vs true {INNER_PER_BYTE} ns/byte")
check("innerHash fit explains the data", r2 > 0.999, f"R2={r2:.4f}")
# The model is linear, so it must also hold far outside the sampled grid.
# Integer flooring in chargeNativeGas loses under 64ns per call, hence 1.001.
big = 1 << 20
inner_charge = charged(round(base), s1_per_1024, s2_per_1024, big, big)
check("innerHash charge covers 1MiB per side",
      inner_truth(big, big) / inner_charge <= 1.001,
      f"charged {inner_charge:.0f} vs cost {inner_truth(big, big):.0f}")

# --- modExp: the cost is superlinear, so the emitted row must not undercharge.
grid = grid_of(MODEXP_GRID, modexp_truth)
base, s1, s2, r2, separable = G.fit_2arg(grid)
s1_per_1024, s2_per_1024 = round(s1 * 1024), round(s2 * 1024)
ratio, wn1, wn2 = G.worst_undercharge(grid, round(base), s1_per_1024,
                                      s2_per_1024)
check("modExp emitted row covers its own worst sampled point", ratio <= 1.0,
      f"undercharges exp={wn1} mod={wn2} by {ratio:.1f}x (R2={r2:.3f})")
# The row the fitter emits must not be cheaper than the row it replaces at the
# modulus size that row was calibrated against.
shipped = 58000 + 24647680 * 256 // 1024
check("modExp stays at least as expensive as the shipped row at mod=256",
      charged(round(base), s1_per_1024, s2_per_1024, 256, 256) >= shipped,
      f"fitted {charged(round(base), s1_per_1024, s2_per_1024, 256, 256):.0f}"
      f" vs shipped {shipped}")

print()
if failures:
    print(f"{len(failures)} failing: " + ", ".join(failures))
    sys.exit(1)
print("all checks passed")
