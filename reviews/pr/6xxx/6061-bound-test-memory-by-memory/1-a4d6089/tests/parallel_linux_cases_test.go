/* Run: from a gno checkout, after threading a root prefix through
ReadMemInfo, cgroupMemory and cgroupDir as the finding describes:
gh pr checkout 6061 -R gnolang/gno && git checkout a4d6089
curl -fsSL -o tm2/pkg/testutils/parallel_linux_cases_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6061-bound-test-memory-by-memory/1-a4d6089/tests/parallel_linux_cases_test.go
go test -v -run 'TestCgroupDir|TestReadMemInfoMeminfo|TestReadMemInfoClampsToCgroup|TestReadMemInfoUnlimitedCgroup' ./tm2/pkg/testutils/
rm tm2/pkg/testutils/parallel_linux_cases_test.go
*/

// Covers the cgroup lookup and the /proc/meminfo parser, both of which bake
// their paths in as literals at a4d6089 and are therefore unreachable.
// Deleting the host-namespace fallback in cgroupDir leaves the suite green at
// that commit and fails TestCgroupDir here.

package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRoot builds a /proc and /sys/fs/cgroup tree under a temp dir. Keys are
// paths relative to the root, so "proc/meminfo" and "sys/fs/cgroup/memory.max".
func fakeRoot(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	return root
}

func TestCgroupDir(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		files map[string]string
		want  string // relative to the root, "" when ok is false
		ok    bool
	}{
		{
			// Under a cgroup namespace the unified path is already the mount.
			name: "namespaced, path is the mount",
			files: map[string]string{
				"proc/self/cgroup":         "0::/\n",
				"sys/fs/cgroup/memory.max": "max\n",
			},
			want: "sys/fs/cgroup", ok: true,
		},
		{
			name: "nested path exists under the mount",
			files: map[string]string{
				"proc/self/cgroup":                              "0::/user.slice/app.scope\n",
				"sys/fs/cgroup/memory.max":                      "max\n",
				"sys/fs/cgroup/user.slice/app.scope/memory.max": "max\n",
			},
			want: "sys/fs/cgroup/user.slice/app.scope", ok: true,
		},
		{
			// A container sharing the host's cgroup namespace: the unified
			// path names a directory that does not exist under the mount this
			// process sees, and the mount root is the reading that applies.
			name: "host namespace, nested path absent, falls back to the mount",
			files: map[string]string{
				"proc/self/cgroup":         "0::/system.slice/docker-deadbeef.scope\n",
				"sys/fs/cgroup/memory.max": "max\n",
			},
			want: "sys/fs/cgroup", ok: true,
		},
		{
			name: "no memory controller anywhere",
			files: map[string]string{
				"proc/self/cgroup":           "0::/system.slice/docker-deadbeef.scope\n",
				"sys/fs/cgroup/cgroup.procs": "\n",
			},
			ok: false,
		},
		{
			// v1 lines name a controller; there is no unified entry to use.
			name: "cgroup v1 only",
			files: map[string]string{
				"proc/self/cgroup":         "12:memory:/user.slice\n11:cpu,cpuacct:/user.slice\n",
				"sys/fs/cgroup/memory.max": "max\n",
			},
			ok: false,
		},
		{
			name:  "proc file absent",
			files: map[string]string{"sys/fs/cgroup/memory.max": "max\n"},
			ok:    false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := fakeRoot(t, tt.files)
			dir, ok := cgroupDir(root)
			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, filepath.Join(root, tt.want), dir)
			}
		})
	}
}

const sampleMeminfo = `MemTotal:       32950272 kB
MemFree:         1048576 kB
MemAvailable:   20971520 kB
Buffers:          123456 kB
`

func TestReadMemInfoMeminfo(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name              string
		meminfo           string
		wantTotal, wantAv uint64
		ok                bool
	}{
		{
			name: "kB fields become bytes", meminfo: sampleMeminfo,
			wantTotal: 32950272 << 10, wantAv: 20971520 << 10, ok: true,
		},
		{
			// Kernels before 3.14 omit MemAvailable. Zero must read as no
			// figure, not as no memory free.
			name:    "no MemAvailable line",
			meminfo: "MemTotal:       32950272 kB\nMemFree:         1048576 kB\n",
			ok:      false,
		},
		{
			name:    "no MemTotal line",
			meminfo: "MemAvailable:   20971520 kB\n",
			ok:      false,
		},
		{
			name:      "malformed values are skipped, not fatal",
			meminfo:   "MemTotal:\nMemTotal:       32950272 kB\nMemAvailable: nonsense kB\nMemAvailable: 4 kB\n",
			wantTotal: 32950272 << 10, wantAv: 4 << 10, ok: true,
		},
		{name: "empty file", meminfo: "", ok: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := fakeRoot(t, map[string]string{"proc/meminfo": tt.meminfo})
			mi, ok := readMemInfo(root)
			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.wantTotal, mi.Total)
				assert.Equal(t, tt.wantAv, mi.Available)
			}
		})
	}

	t.Run("meminfo absent", func(t *testing.T) {
		t.Parallel()
		_, ok := readMemInfo(fakeRoot(t, nil))
		assert.False(t, ok)
	})
}

func TestReadMemInfoClampsToCgroup(t *testing.T) {
	t.Parallel()

	root := fakeRoot(t, map[string]string{
		"proc/meminfo":                 sampleMeminfo,
		"proc/self/cgroup":             "0::/\n",
		"sys/fs/cgroup/memory.max":     "4294967296\n", // 4 GiB
		"sys/fs/cgroup/memory.current": "1073741824\n", // 1 GiB
	})
	mi, ok := readMemInfo(root)
	require.True(t, ok)
	assert.Equal(t, uint64(4<<30), mi.Total, "host total must narrow to the cgroup limit")
	assert.Equal(t, uint64(3<<30), mi.Available)
}

func TestReadMemInfoUnlimitedCgroup(t *testing.T) {
	t.Parallel()

	// "max" is the common case; the host reading must survive it untouched.
	root := fakeRoot(t, map[string]string{
		"proc/meminfo":                 sampleMeminfo,
		"proc/self/cgroup":             "0::/\n",
		"sys/fs/cgroup/memory.max":     "max\n",
		"sys/fs/cgroup/memory.current": "1073741824\n",
	})
	mi, ok := readMemInfo(root)
	require.True(t, ok)
	assert.Equal(t, uint64(32950272<<10), mi.Total)
	assert.Equal(t, uint64(20971520<<10), mi.Available)
}
