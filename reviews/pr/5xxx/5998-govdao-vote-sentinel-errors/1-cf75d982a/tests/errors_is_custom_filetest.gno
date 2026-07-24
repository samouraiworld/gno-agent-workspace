/* Run: from a gno checkout:
gh pr checkout 5998 -R gnolang/gno && git checkout cf75d982a
curl -fsSL -o gnovm/tests/files/errors_is_custom_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5998-govdao-vote-sentinel-errors/1-cf75d982a/tests/errors_is_custom_filetest.gno
go test -v -run 'TestFiles/errors_is_custom_filetest.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/errors_is_custom_filetest.gno
*/

// Mirrors the ProposalClosedError shape against gno's errors package: a
// concrete pointer type carrying dynamic content plus an Is(error) bool
// matching one sentinel. The golden is the byte-identical output of the same
// program built with the Go toolchain (errors_is_custom.go, go1.26.5).

package main

import (
	"errors"
	"fmt"
)

var (
	ErrProposalClosed   = errors.New("proposal closed")
	ErrProposalNotFound = errors.New("proposal not found")
)

type ProposalClosedError struct{ Accepted bool }

func (e *ProposalClosedError) Error() string {
	return fmt.Sprintf("proposal closed. Accepted: %v", e.Accepted)
}

func (e *ProposalClosedError) Is(target error) bool { return target == ErrProposalClosed }

func main() {
	var err error = &ProposalClosedError{Accepted: true}
	fmt.Println("msg:", err.Error())
	fmt.Println("is-closed:", errors.Is(err, ErrProposalClosed))
	fmt.Println("is-notfound:", errors.Is(err, ErrProposalNotFound))
	fmt.Println("is-self:", errors.Is(err, err))
	fmt.Println("reverse:", errors.Is(ErrProposalClosed, err))
	fmt.Println("nil-err:", errors.Is(nil, ErrProposalClosed))
	fmt.Println("unwrap:", errors.Unwrap(err))
}

// Output:
// msg: proposal closed. Accepted: true
// is-closed: true
// is-notfound: false
// is-self: true
// reverse: false
// nil-err: false
// unwrap: <nil>
