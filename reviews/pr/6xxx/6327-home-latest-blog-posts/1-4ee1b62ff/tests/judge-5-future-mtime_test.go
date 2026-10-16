package gnoweb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A static file whose mtime is an hour ahead of the clock: every resolve
// re-reads it, since modTime is zeroed after each read. At 4ee1b62ff each
// resolve returns the new same-size content and logs modTimeZero=true.
func TestZZJudge5FutureMtimeRereads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home.md")
	future := time.Now().Add(time.Hour)
	if err := os.WriteFile(path, []byte("# A"), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(path, future, future)
	target, err := NewStaticFileAlias(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		os.WriteFile(path, []byte{'#', ' ', byte('B' + i)}, 0o644) // same size, same mtime
		os.Chtimes(path, future, future)
		target = target.resolve()
		t.Logf("resolve %d: Value=%q modTimeZero=%v", i, target.Value, target.file.modTime.IsZero())
	}
}
