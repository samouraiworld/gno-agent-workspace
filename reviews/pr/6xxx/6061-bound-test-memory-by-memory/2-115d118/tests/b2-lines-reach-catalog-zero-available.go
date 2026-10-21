//go:build linux

// Probe for PR 6061 round 2: what ReadMemInfo returns inside a cgroup v2 scope
// whose memory.max is under pressure.
//
// Repro from a plain clone of gnolang/gno at 115d118c7, on a cgroup v2 host
// with a systemd user manager:
//
//	cp <this file> tm2/pkg/testutils/zz_pressure_test.go
//	go test -c -o /tmp/tu.test ./tm2/pkg/testutils
//	# anon: allocate past memory.max with swap allowed, count ok=false readings
//	ZZ_MODE=anon systemd-run --user --scope -q -p MemoryMax=256M -p MemorySwapMax=2G \
//	  /tmp/tu.test -test.run TestZZPressure -test.v
//	# cache: page cache read twice, then anon allocation on top with no swap
//	ZZ_MODE=cache ZZ_DIR=$PWD systemd-run --user --scope -q -p MemoryMax=512M -p MemorySwapMax=0 \
//	  /tmp/tu.test -test.run TestZZPressure -test.v
package testutils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func zzRaw(dir string) string {
	r := func(name string) string {
		b, _ := os.ReadFile(filepath.Join(dir, name))
		return strings.TrimSpace(string(b))
	}
	act, _ := readCgroupStat(filepath.Join(dir, "memory.stat"), "active_file")
	inact, _ := readCgroupStat(filepath.Join(dir, "memory.stat"), "inactive_file")
	return fmt.Sprintf("max=%s current=%s active_file=%d inactive_file=%d", r("memory.max"), r("memory.current"), act, inact)
}

func zzTouch(n int) []byte {
	b := make([]byte, n)
	for j := 0; j < len(b); j += 4096 {
		b[j] = 1
	}
	return b
}

func TestZZPressure(t *testing.T) {
	mode := os.Getenv("ZZ_MODE")
	if mode == "" {
		t.Skip("probe; set ZZ_MODE")
	}
	cg, _ := os.ReadFile("/proc/self/cgroup")
	rel, _ := parseProcCgroup(string(cg))
	dir := filepath.Join(cgroupMount, rel)
	t.Logf("cgroup %s", rel)

	switch mode {
	case "anon":
		var hold [][]byte
		readings, notOK := 0, 0
		minAvail := ^uint64(0)
		for i := 1; i <= 80; i++ { // 80 x 8 MiB = 640 MiB against a 256 MiB limit
			hold = append(hold, zzTouch(8<<20))
			for k := 0; k < 25; k++ {
				mi, ok := ReadMemInfo()
				readings++
				if !ok {
					notOK++
				} else if mi.Available < minAvail {
					minAvail = mi.Available
				}
			}
			if i%10 == 0 {
				mi, ok := ReadMemInfo()
				t.Logf("held=%dMiB ok=%v Available=%d %s", i*8, ok, mi.Available, zzRaw(dir))
			}
		}
		t.Logf("readings=%d ok=false=%d minAvailable(ok)=%d", readings, notOK, minAvail)
		_ = hold
	case "cache":
		path := filepath.Join(os.Getenv("ZZ_DIR"), "zz-cache.bin")
		defer os.Remove(path)
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		chunk := make([]byte, 1<<20)
		for i := range chunk {
			chunk[i] = byte(i)
		}
		for i := 0; i < 320; i++ {
			f.Write(chunk)
		}
		f.Sync()
		f.Close()
		for pass := 0; pass < 3; pass++ {
			g, _ := os.Open(path)
			io.Copy(io.Discard, g)
			g.Close()
		}
		mi, ok := ReadMemInfo()
		t.Logf("after reading 320MiB x3: ok=%v Available=%dMiB Total=%dMiB %s", ok, mi.Available>>20, mi.Total>>20, zzRaw(dir))
		// Take more anon memory than Available claimed is left, with no swap.
		want := int(mi.Available) + 128<<20
		hold := zzTouch(want)
		mi2, ok2 := ReadMemInfo()
		t.Logf("then held %dMiB anon with no swap, no OOM: ok=%v Available=%dMiB %s", want>>20, ok2, mi2.Available>>20, zzRaw(dir))
		_ = hold
	}
}
