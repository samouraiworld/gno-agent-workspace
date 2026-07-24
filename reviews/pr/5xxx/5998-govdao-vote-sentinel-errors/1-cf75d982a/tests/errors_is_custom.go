/* Run: from any directory:
curl -fsSL -o /tmp/errors_is_custom/main.go --create-dirs \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5998-govdao-vote-sentinel-errors/1-cf75d982a/tests/errors_is_custom.go
cd /tmp/errors_is_custom && go mod init parity && go run .
*/

// Go side of the parity pair for errors_is_custom_filetest.gno: same program,
// Go toolchain. Its stdout is the filetest's // Output: golden, so a
// divergence between gno's errors.Is and Go's shows up as a filetest failure.

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
