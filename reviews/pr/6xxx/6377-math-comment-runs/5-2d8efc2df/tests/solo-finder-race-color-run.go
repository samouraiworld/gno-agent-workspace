// solo-finder, PR 6377 round 5 at 2d8efc2df: TestBareColorRunIsLinear exceeds its 2 s bound under -race.
//
// Repro from a plain clone of gnolang/gno:
//   git checkout 2d8efc2df67aa2f5da7d8d1b30925178691a0e18
//   cp solo-finder-race-color-run.go gno.land/pkg/gnoweb/markdown/mathml/zz_solo_time_test.go
//   go test -race -count=3 -run 'TestBareColorRunIsLinear|TestZZSoloCallTimes' -v ./gno.land/pkg/gnoweb/markdown/mathml/
//   go test -count=1 -run 'TestZZSoloCallTimes' -v ./gno.land/pkg/gnoweb/markdown/mathml/
//
// Observed (16 cores, go1.27.3): under -race, 3 of 3 runs
//   edgecases_test.go:148: conversion of "\\color[RGB]..." did not terminate / --- FAIL: TestBareColorRunIsLinear (2.66s)
//   per call under -race: \color[RGB] x 2^16 2.485s, \color[{&}] x 2^16 2.315s, \color x 2^17 + comments 1.979s
//   per call without -race: 560ms, 396ms, 512ms. Each run is linear (ns per unit flat from 2^12 to 2^16).
// CI's race job runs only TestMathConcurrentRender in ./gno.land/pkg/gnoweb/markdown/, so no job reaches this.
package mathml

import (
	"strings"
	"testing"
	"time"
)

// Per-call time of each conversion TestBareColorRunIsLinear and TestCommentRunIsLinear bound by 2 s.
func TestZZSoloCallTimes(t *testing.T) {
	inputs := map[string]string{}
	for _, unit := range []string{`\color`, `\color[RGB]`, `\color[{&}]`} {
		inputs[unit+" x 2^16"] = strings.Repeat(unit, 1<<16) + "x"
	}
	inputs[`\color[ x 2^15`] = strings.Repeat(`\color[`, 1<<15) + "x" + strings.Repeat("]", 1<<15)
	inputs[`\color x 2^17 + comments`] = strings.Repeat(`\color`, 1<<17) + strings.Repeat("%\n", 1<<17)
	inputs[`comments 2^17`] = "x" + strings.Repeat("%\n", (1<<18)/2) + "y"
	for name, tex := range inputs {
		start := time.Now()
		NewMathMLConverter().ConvertInline(tex)
		t.Logf("%-28s %v", name, time.Since(start).Round(time.Millisecond))
	}
}
