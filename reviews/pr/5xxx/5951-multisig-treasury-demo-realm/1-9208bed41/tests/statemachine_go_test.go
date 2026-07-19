/* Run: from a gno checkout:
gh pr checkout 5951 -R gnolang/gno && git checkout 9208bed41
mkdir -p /tmp/msparity && cp <this file> /tmp/msparity/statemachine_go_test.go
cd /tmp/msparity && go mod init msparity && go test -v ./...
*/

// Go mirror of the Setup/Propose/Approve/Execute state machine in
// examples/gno.land/r/demo/multisig/multisig.gno, minus the banker call.
// At 9208bed41 the Gno realm and this mirror agree on every case below,
// so the duplicate-owner lockout and the decimal-key render order are
// properties of the realm's own logic, not of Gno diverging from Go.
package msparity

import (
	"sort"
	"strconv"
	"testing"
)

type proposal struct {
	id         int
	to         string
	amount     int64
	executed   bool
	approvals  map[string]bool
	approveCnt int
}

type multisig struct {
	owners      map[string]bool
	ownerCount  int
	threshold   int
	initialized bool
	proposals   map[string]*proposal
	nextID      int
}

func newMultisig() *multisig {
	return &multisig{
		owners:    map[string]bool{},
		proposals: map[string]*proposal{},
		nextID:    1,
	}
}

// Setup mirrors multisig.gno Setup: no distinctness check, ownerCount
// hardcoded to 3, threshold bounded only by the literal 3.
func (m *multisig) Setup(o1, o2, o3 string, thresh int) {
	if m.initialized {
		panic("already initialized")
	}
	if thresh < 1 || thresh > 3 {
		panic("invalid threshold")
	}
	m.owners[o1] = true
	m.owners[o2] = true
	m.owners[o3] = true
	m.ownerCount = 3
	m.threshold = thresh
	m.initialized = true
}

func (m *multisig) assertOwner(caller string) {
	if !m.initialized {
		panic("multisig not initialized")
	}
	if !m.owners[caller] {
		panic("caller is not an owner")
	}
}

// Propose mirrors multisig.gno Propose: the amount is stored unchecked.
func (m *multisig) Propose(caller, to string, amount int64) int {
	m.assertOwner(caller)
	id := m.nextID
	m.nextID++
	m.proposals[strconv.Itoa(id)] = &proposal{
		id: id, to: to, amount: amount, approvals: map[string]bool{},
	}
	return id
}

func (m *multisig) Approve(caller string, id int) {
	m.assertOwner(caller)
	p, ok := m.proposals[strconv.Itoa(id)]
	if !ok {
		panic("proposal does not exist")
	}
	if p.executed {
		panic("proposal already executed")
	}
	if p.approvals[caller] {
		panic("already approved")
	}
	p.approvals[caller] = true
	p.approveCnt++
}

// Execute mirrors multisig.gno Execute, stopping short of the send.
// Note there is no owner check on this entry point.
func (m *multisig) Execute(id int) int64 {
	p, ok := m.proposals[strconv.Itoa(id)]
	if !ok {
		panic("proposal does not exist")
	}
	if p.executed {
		panic("proposal already executed")
	}
	if p.approveCnt < m.threshold {
		panic("not enough approvals")
	}
	p.executed = true
	return p.amount
}

func mustPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic %q, got none", want)
		}
		if got, _ := r.(string); got != want {
			t.Fatalf("expected panic %q, got %q", want, got)
		}
	}()
	fn()
}

// Three identical owners are accepted, collapse to one tree entry, and
// leave ownerCount reporting 3. Any threshold above 1 is then
// unreachable, so a funded treasury can never pay out.
func TestDuplicateOwnersLockTheTreasury(t *testing.T) {
	m := newMultisig()
	m.Setup("owner", "owner", "owner", 2)

	if m.ownerCount != 3 {
		t.Fatalf("ownerCount = %d, want 3", m.ownerCount)
	}
	if len(m.owners) != 1 {
		t.Fatalf("distinct owners = %d, want 1", len(m.owners))
	}

	id := m.Propose("owner", "recipient", 100)
	m.Approve("owner", id)
	mustPanic(t, "already approved", func() { m.Approve("owner", id) })
	mustPanic(t, "not enough approvals", func() { m.Execute(id) })
}

// Setup should reject a threshold no reachable owner set can satisfy.
func TestSetupRejectsUnreachableThreshold(t *testing.T) {
	m := newMultisig()
	mustPanic(t, "invalid threshold", func() { m.Setup("a", "a", "b", 3) })
}

// Propose should reject a payout the bank keeper will refuse anyway.
func TestProposeRejectsNonPositiveAmount(t *testing.T) {
	m := newMultisig()
	m.Setup("a", "b", "c", 2)
	mustPanic(t, "invalid amount", func() { m.Propose("a", "d", -500) })
	mustPanic(t, "invalid amount", func() { m.Propose("a", "d", 0) })
}

// Execute is callable by a non-owner once the threshold is met.
func TestExecuteNeedsNoOwner(t *testing.T) {
	m := newMultisig()
	m.Setup("a", "b", "c", 2)
	id := m.Propose("a", "d", 100)
	m.Approve("a", id)
	m.Approve("b", id)
	if got := m.Execute(id); got != 100 {
		t.Fatalf("payout = %d, want 100", got)
	}
}

// Decimal-string keys sort lexicographically in Go's sort and in the
// realm's avl tree alike, so Render lists 1, 10, 11, 2 in both.
func TestDecimalKeyOrderMatchesGno(t *testing.T) {
	m := newMultisig()
	m.Setup("a", "b", "c", 2)
	for i := 0; i < 11; i++ {
		m.Propose("a", "d", 1)
	}
	keys := make([]string, 0, len(m.proposals))
	for k := range m.proposals {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	want := []string{"1", "10", "11", "2", "3", "4", "5", "6", "7", "8", "9"}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("key[%d] = %s, want %s", i, keys[i], want[i])
		}
	}
	if keys[1] != "10" {
		t.Fatalf("expected proposal 10 to sort before proposal 2")
	}
}
