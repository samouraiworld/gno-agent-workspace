package cstypes_test

// gnolang/gno#6152, solo finder probe: amino.DeepCopy slices, callers and siblings.
//
// Repro from a plain clone of github.com/gnolang/gno (Go 1.25):
//   git checkout a742126999443305c6bfcf6fd6364ea57fba51d4
//   cp solo-finder-deepcopy-probe.go tm2/pkg/bft/consensus/types/zz_solofinder_test.go
//   go test ./tm2/pkg/bft/consensus/types -run TestSolo -v -count=1
//   git checkout 7916d1dd65f326efe46b5ad105412ce028768c4d   # merge base, same file, same command
//
// Observed at head a74212699:
//   TestSoloTypeWalk: no UNEXP-UNDER-SLICE and no MAP line under any caller type
//   header hash src==copy: true; AppHash(empty) nil copy=false; DataHash(nil) nil copy=true
//   header hash AppHash nil vs empty equal: true; params hash src==copy: true
//   header copy sees write to source: false
//   *[]int nil: copy nil=true; *map nil: copy nil=false
//   **int nil: PANIC unsupported type invalid
//   *any nil: PANIC reflect: call of reflect.Value.Type on zero Value
//   sdk.Context{}.BlockHeader(): PANIC reflect: call of reflect.Value.Type on zero Value
//   map value aliased: true; unexported field in slice element kept: y=0
// Observed at merge base 7916d1dd6: identical except
//   header copy sees write to source: true; unexported field in slice element kept: y=2

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/amino"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	cnscfg "github.com/gnolang/gno/tm2/pkg/bft/consensus/config"
	cstypes "github.com/gnolang/gno/tm2/pkg/bft/consensus/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto/multisig"
	"github.com/gnolang/gno/tm2/pkg/sdk"
)

var soloTimeT = reflect.TypeOf(time.Time{})

// soloWalk mirrors _deepCopy's traversal over types and reports, for every
// type reachable from a DeepCopy caller, the sites whose copy loses data.
func soloWalk(t reflect.Type, path string, inSlice bool, seen map[string]bool, out map[string]bool) {
	key := fmt.Sprintf("%v/%v", t, inSlice)
	if seen[key] {
		return
	}
	seen[key] = true
	if m, ok := t.MethodByName("DeepCopy"); ok && m.Type.NumIn() == 1 && m.Type.NumOut() == 1 {
		out["HOOK DeepCopy "+path] = true
		return
	}
	pt := t
	if t.Kind() != reflect.Pointer {
		pt = reflect.PointerTo(t)
	}
	if _, ok := pt.MethodByName("MarshalAmino"); ok {
		if _, ok2 := pt.MethodByName("UnmarshalAmino"); ok2 {
			out["HOOK amino "+path] = true
			return
		}
	}
	switch t.Kind() {
	case reflect.Pointer:
		soloWalk(t.Elem(), path, inSlice, seen, out)
	case reflect.Slice:
		soloWalk(t.Elem(), path+"[]", true, seen, out)
	case reflect.Array:
		soloWalk(t.Elem(), path+"[N]", inSlice, seen, out)
	case reflect.Map:
		out[fmt.Sprintf("MAP %s %v", path, t)] = true
	case reflect.Interface:
		if inSlice {
			out[fmt.Sprintf("IFACE-UNDER-SLICE %s %v", path, t)] = true
		}
	case reflect.Struct:
		if t == soloTimeT {
			return
		}
		var unexp []string
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				unexp = append(unexp, f.Name)
				continue
			}
			soloWalk(f.Type, path+"."+f.Name, inSlice, seen, out)
		}
		if len(unexp) > 0 {
			tag := "UNEXP"
			if inSlice {
				tag = "UNEXP-UNDER-SLICE"
			}
			out[fmt.Sprintf("%s %s %v %v", tag, path, t, unexp)] = true
		}
	}
}

func TestSoloTypeWalk(t *testing.T) {
	roots := map[string]reflect.Type{
		"RoundState":        reflect.TypeOf(cstypes.RoundState{}),
		"Header":            reflect.TypeOf(&bft.Header{}),
		"ConsensusParams":   reflect.TypeOf(&abci.ConsensusParams{}),
		"ConsensusConfig":   reflect.TypeOf(&cnscfg.ConsensusConfig{}),
		"Evidence(dyn)":     reflect.TypeOf(&bft.DuplicateVoteEvidence{}),
		"MultisigKey(dyn)":  reflect.TypeOf(multisig.PubKeyMultisigThreshold{}),
	}
	names := make([]string, 0, len(roots))
	for n := range roots {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		out := map[string]bool{}
		soloWalk(roots[n], n, n == "Evidence(dyn)", map[string]bool{}, out)
		lines := make([]string, 0, len(out))
		for l := range out {
			lines = append(lines, l)
		}
		sort.Strings(lines)
		for _, l := range lines {
			t.Logf("%s", l)
		}
	}
}

func TestSoloHashes(t *testing.T) {
	h := &bft.Header{
		ChainID: "c", Height: 3, ValidatorsHash: []byte{7},
		AppHash: []byte{}, LastCommitHash: []byte{1, 2}, DataHash: nil,
	}
	c := h.Copy()
	t.Logf("header hash src==copy: %v", bytes.Equal(h.Hash(), c.Hash()))
	t.Logf("AppHash(empty) nil src=%v copy=%v; DataHash(nil) nil src=%v copy=%v",
		h.AppHash == nil, c.AppHash == nil, h.DataHash == nil, c.DataHash == nil)
	h2 := *h
	h2.AppHash = nil
	t.Logf("header hash AppHash nil vs empty equal: %v", bytes.Equal(h.Hash(), h2.Hash()))
	h.LastCommitHash[0] = 9
	t.Logf("header copy sees write to source: %v", c.LastCommitHash[0] == 9)

	cp := abci.ConsensusParams{
		Block:     &abci.BlockParams{MaxGas: 5},
		Validator: &abci.ValidatorParams{PubKeyTypeURLs: []string{}},
	}
	cc := amino.DeepCopy(&cp).(*abci.ConsensusParams)
	t.Logf("params hash src==copy: %v; PubKeyTypeURLs nil src=%v copy=%v",
		bytes.Equal(cp.Hash(), cc.Hash()), cp.Validator.PubKeyTypeURLs == nil, cc.Validator.PubKeyTypeURLs == nil)
	cp2 := cp
	cp2.Validator = &abci.ValidatorParams{}
	t.Logf("params hash PubKeyTypeURLs nil vs empty equal: %v", bytes.Equal(cp.Hash(), cp2.Hash()))
}

type soloU struct {
	X int
	y int
}

type soloMS struct {
	M map[string][]byte
}

func TestSoloSiblings(t *testing.T) {
	try := func(name string, f func() string) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("%s: PANIC %v", name, r)
			}
		}()
		t.Logf("%s: %s", name, f())
	}
	var s []int
	try("*[]int nil", func() string { return fmt.Sprintf("copy nil=%v", *amino.DeepCopy(&s).(*[]int) == nil) })
	var m map[string]int
	try("*map nil", func() string { return fmt.Sprintf("copy nil=%v", *amino.DeepCopy(&m).(*map[string]int) == nil) })
	var p *int
	try("**int nil", func() string { return fmt.Sprintf("copy=%v", amino.DeepCopy(&p)) })
	var i any
	try("*any nil", func() string { return fmt.Sprintf("copy=%v", amino.DeepCopy(&i)) })
	try("sdk.Context{}.BlockHeader()", func() string { return fmt.Sprintf("%v", sdk.Context{}.BlockHeader()) })

	src := soloMS{M: map[string][]byte{"a": {1}}}
	cpy := amino.DeepCopy(src).(soloMS)
	src.M["a"][0] = 9
	try("map value aliased", func() string { return fmt.Sprintf("%v", cpy.M["a"][0] == 9) })

	us := []soloU{{X: 1, y: 2}}
	uc := amino.DeepCopy(us).([]soloU)
	try("unexported field in slice element kept", func() string { return fmt.Sprintf("y=%d", uc[0].y) })
}
