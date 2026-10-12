package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeT struct{ failed bool }

func (f *fakeT) Errorf(string, ...any) { f.failed = true }

func TestJudgeErrorContainsNil(t *testing.T) {
	f := &fakeT{}
	ok := assert.ErrorContains(f, nil, "InitialHeight mismatch")
	t.Logf("ErrorContains(nil) ok=%v failed=%v", ok, f.failed)
	if ok || !f.failed {
		t.Fatal("ErrorContains passed a nil error")
	}
}
