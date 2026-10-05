# Review: [#6271](https://github.com/gnolang/gno/pull/6271)
Posted: https://github.com/gnolang/gno/pull/6271#pullrequestreview-5416998028
Event: COMMENT
Verdict: APPROVE. The code holds every strictness rule and cap it states; two Missing tests leave the CIDv0/CIDv1 discriminators and the gas-only caps in `Parse` and `DecodeFirst` unpinned, and neither blocks.
Model: claude-opus-5-5, finders xhigh, judge and writer high, standard review, solo wider
Commit: 2a84c1dcd
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6271 2a84c1dcd`
Round: 1. Two finders, one reflector pass, 3 candidates: 2 from the finders, each run from scratch by a judge that was not its finder, and 1 from the reflector, run by the judge that raised it; the finders settled 14 more on their own read.

## Body

> AI review, claude-opus-5-5, standard review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6271-ipfs-content-identifiers/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6271-ipfs-content-identifiers/1-2a84c1d/claims.md) · Status: APPROVE

## examples/gno.land/p/omarsy/cid/v0/cid.gno:32 [gh](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/cid.gno#L32) · [↗](../../../../../.worktrees/gno-review-6271/examples/gno.land/p/omarsy/cid/v0/cid.gno#L32) · Missing test [posted](https://github.com/gnolang/gno/pull/6271#discussion_r4185805041)

Missing test: no test fails when this `len(s) == 46` bound is relaxed to `len(s) >= 2` or the `maxCIDLen` truncation in [`DecodeFirst`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/cid.gno#L66-L68) · [↗](../../../../../.worktrees/gno-review-6271/examples/gno.land/p/omarsy/cid/v0/cid.gno#L66) is removed.

<details><summary>test cases</summary>

`filetests/z_caps_filetest.gno`:

```go
package main

import (
	"strings"

	cid "gno.land/p/omarsy/cid/v0"
)

func main() {
	_, err := cid.Parse("Qm" + strings.Repeat("z", 510))
	println(err)
	_, n, _ := cid.DecodeFirst(make([]byte, 1<<16))
	println(n)
}

// Output:
// cid: invalid multibase string
// 0

// Gas:
// 760183
```

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6271 -R gnolang/gno
go install ./gnovm/cmd/gno
cd examples/gno.land/p/omarsy/cid/v0
cat > filetests/z_caps_filetest.gno <<'EOF'
package main

import (
	"strings"

	cid "gno.land/p/omarsy/cid/v0"
)

func main() {
	_, err := cid.Parse("Qm" + strings.Repeat("z", 510))
	println(err)
	_, n, _ := cid.DecodeFirst(make([]byte, 1<<16))
	println(n)
}

// Output:
// cid: invalid multibase string
// 0

// Gas:
// 760183
EOF
echo "head:"; gno test . 2>&1 | grep -E '^ok|FAIL: ./z_caps'
perl -pi -e 's/len\(s\) == 46 && s/len(s) >= 2 && s/' cid.gno
echo "bound relaxed:"; gno test . 2>&1 | grep -E '^ok|FAIL: ./z_caps'
git checkout -- cid.gno
perl -0pi -e 's/\tif len\(b\) > maxCIDLen \{\n\t\tb = b\[:maxCIDLen\]\n\t\}\n//' cid.gno
echo "truncation removed:"; gno test . 2>&1 | grep -E '^ok|FAIL: ./z_caps'
git checkout -- cid.gno && rm filetests/z_caps_filetest.gno
```

The filetest passes at the head and is the only failure under each edit:

```text
head:
ok      . 	3.65s
bound relaxed:
--- FAIL: ./z_caps_filetest.gno (elapsed: 0.02s, gas: 37477045, storage: gno.land/p/omarsy/cid/v0:+30697b)
truncation removed:
--- FAIL: ./z_caps_filetest.gno (elapsed: 0.02s, gas: 1027783, storage: gno.land/p/omarsy/cid/v0:+30697b)
```

</details>

## examples/gno.land/p/omarsy/cid/v0/cid.gno:116 [gh](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/cid.gno#L116) · [↗](../../../../../.worktrees/gno-review-6271/examples/gno.land/p/omarsy/cid/v0/cid.gno#L116) · Missing test [posted](https://github.com/gnolang/gno/pull/6271#discussion_r4185805053)

Missing test: no test fails when this `b[0] == 0x12` check is dropped, though `DecodeFirst` then returns the first 34 bytes of a codec-`0x20` CIDv1 as a CIDv0 with no error.

<details><summary>test cases</summary>

No CIDv1 in the tests has `0x20` as its second byte, and every 46-character string they parse starts with `Qm`, so two checks go unpinned:

- without `b[0] == 0x12`, `DecodeFirst` reads a CIDv1 with codec `0x20` (`01 20 12 20 …`) as a 34-byte CIDv0 and returns no error, and `Decode` rejects that CIDv1;
- without `s[:2] == "Qm"` in [`Parse`](https://github.com/gnolang/gno/blob/2a84c1dcdbd91a15a18923f56180b6367de97e21/examples/gno.land/p/omarsy/cid/v0/cid.gno#L32) · [↗](../../../../../.worktrees/gno-review-6271/examples/gno.land/p/omarsy/cid/v0/cid.gno#L32), a 46-character base32 CIDv1, such as a `Raw` CID over a 24-byte identity digest, goes to the base58btc decoder and is rejected.

```go
func TestDecodeCodec0x20(t *testing.T) {
	// A CIDv1 whose codec varint is 0x20: 01 20 12 20 <32 bytes>.
	c, err := NewV1(0x20, Sum(Raw, []byte("hello")).Multihash())
	urequire.NoError(t, err)
	d, err := Decode(c.Bytes())
	urequire.NoError(t, err)
	uassert.True(t, d == c)
	f, n, err := DecodeFirst(c.Bytes())
	urequire.NoError(t, err)
	uassert.Equal(t, 36, n)
	uassert.True(t, f == c)
}

func TestParseBase32Len46(t *testing.T) {
	// Raw and a 24-byte identity digest: 28 bytes, 46 characters in base32.
	mh, err := NewMultihash(Identity, []byte("abcdefghijklmnopqrstuvwx"))
	urequire.NoError(t, err)
	c, err := NewV1(Raw, mh)
	urequire.NoError(t, err)
	uassert.Equal(t, 46, len(c.String()))
	p, err := Parse(c.String())
	urequire.NoError(t, err)
	uassert.True(t, p == c)
}
```

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6271 -R gnolang/gno
go install ./gnovm/cmd/gno
cd examples/gno.land/p/omarsy/cid/v0
cat > zz_discriminators_test.gno <<'EOF'
package cid

import (
	"testing"

	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestDecodeCodec0x20(t *testing.T) {
	c, err := NewV1(0x20, Sum(Raw, []byte("hello")).Multihash())
	urequire.NoError(t, err)
	d, err := Decode(c.Bytes())
	urequire.NoError(t, err)
	uassert.True(t, d == c)
	f, n, err := DecodeFirst(c.Bytes())
	urequire.NoError(t, err)
	uassert.Equal(t, 36, n)
	uassert.True(t, f == c)
}

func TestParseBase32Len46(t *testing.T) {
	mh, err := NewMultihash(Identity, []byte("abcdefghijklmnopqrstuvwx"))
	urequire.NoError(t, err)
	c, err := NewV1(Raw, mh)
	urequire.NoError(t, err)
	uassert.Equal(t, 46, len(c.String()))
	p, err := Parse(c.String())
	urequire.NoError(t, err)
	uassert.True(t, p == c)
}
EOF
echo "head:"; gno test . 2>&1 | grep -E '^ok|--- FAIL'
perl -pi -e 's/len\(b\) == 34 && b\[0\] == 0x12 && /len(b) == 34 && /' cid.gno
mv zz_discriminators_test.gno zz.bak
echo "b[0] check dropped, suite:"; gno test . 2>&1 | grep -E '^ok|--- FAIL'
mv zz.bak zz_discriminators_test.gno; echo "b[0] check dropped, new tests:"; gno test . 2>&1 | grep -E '^ok|--- FAIL'
git checkout -- cid.gno
perl -pi -e 's/len\(s\) == 46 && s\[:2\] == "Qm"/len(s) == 46/' cid.gno
mv zz_discriminators_test.gno zz.bak
echo "Qm check dropped, suite:"; gno test . 2>&1 | grep -E '^ok|--- FAIL'
mv zz.bak zz_discriminators_test.gno
echo "Qm check dropped, new tests:"; gno test . 2>&1 | grep -E '^ok|--- FAIL'
git checkout -- cid.gno && rm zz_discriminators_test.gno
```

The new tests pass at the head and fail under each edit, while the package's own suite stays green:

```text
head:
ok      . 	3.63s
b[0] check dropped, suite:
ok      . 	3.68s
b[0] check dropped, new tests:
--- FAIL: TestDecodeCodec0x20 (0.00s)
Qm check dropped, suite:
ok      . 	3.66s
Qm check dropped, new tests:
--- FAIL: TestParseBase32Len46 (0.00s)
```

</details>
