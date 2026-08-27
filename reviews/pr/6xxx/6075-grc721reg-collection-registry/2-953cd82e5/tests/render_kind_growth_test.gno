/* Run: from a gno checkout, green at 953cd82e5. Each test prints one GAS line;
the render cost is the Render figure minus its Fill twin.
gh pr checkout 6075 -R gnolang/gno && git checkout 953cd82e5
P=examples/gno.land/r/demo/defi/grc721reg
curl -fsSL -o $P/zprobe_test.gno https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6075-grc721reg-collection-registry/2-953cd82e5/tests/render_kind_growth_test.gno
cd $P
for t in TestZFillOnly0 TestZFillRender0 TestZFillOnly2k TestZFillRender2k TestZOutOfRangePage TestZSetBeforeChecks; do
  printf '%-22s ' $t; gno test -v -run "$t\$" . 2>&1 | grep -- '--- GAS' | sort -t: -k2 -rn | head -1
done
cd - && rm $P/zprobe_test.gno
*/

package grc721reg

import (
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/collection"
	"gno.land/p/nt/ufmt/v0"
	"gno.land/p/nt/urequire/v0"
)

type vView struct{ k, id string }

func (v *vView) ExtensionKind() string { return v.k }
func (v *vView) TokenID() string       { return v.id }

// One whole index page of entries, each grown after registration so the
// maxEventKinds ceiling Register enforces is already behind us.
func fillPage(cur realm, kinds int) {
	for e := 0; e < pageSize; e++ {
		testing.SetRealm(testing.NewCodeRealm(ufmt.Sprintf("gno.land/r/demo/zv%02d", e)))

		nft, _ := collection.New("E", ufmt.Sprintf("E%02d", e), 0, cur)
		Register(cross(cur), nft.Collection(), "")

		for j := 0; j < kinds; j++ {
			nft.Attach(&vView{k: ufmt.Sprintf("k%05d", j), id: nft.Token().ID()})
		}
	}
}

func TestZFillOnly0(cur realm, t *testing.T) { fillPage(cur, 0) }

func TestZFillRender0(cur realm, t *testing.T) {
	fillPage(cur, 0)
	_ = Render("")
}

func TestZFillOnly2k(cur realm, t *testing.T) { fillPage(cur, 2000) }

func TestZFillRender2k(cur realm, t *testing.T) {
	fillPage(cur, 2000)
	t.Log("page bytes:", len(Render("")))
}

// A page past the last one.
func TestZOutOfRangePage(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/zoob"))
	token, _ := grc721.NewToken("Oob", "OOB", 0, cur)
	Register(cross(cur), collection.NewCollection(token), "")

	out := Render("?page=999")
	t.Log("items rendered:", strings.Count(out, "info"), "| body:", strings.TrimSpace(out))
}

// A registration Register refused, still in the tree.
func TestZSetBeforeChecks(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/zset"))
	token, _ := grc721.NewToken("Set", "SET", 0, cur)
	long := &vView{k: strings.Repeat("k", maxKindLen+1), id: token.ID()}

	urequire.AbortsContains(t, cur, "extension kind too long", func() {
		Register(cross(cur), collection.NewCollection(token, long), "")
	})

	t.Log("Register refused it, registry holds it:", Get("gno.land/r/demo/zset.SET") != nil,
		"| index renders it:", strings.Contains(Render(""), "Set"))
}
