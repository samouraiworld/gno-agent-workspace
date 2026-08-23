// PKGPATH: gno.land/r/decltypemutual
/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/tests/files/zdecltype_mutual_realm.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/consensus/zdecltype_mutual_realm.gno
go test -v -run 'TestFiles/zdecltype_mutual_realm.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zdecltype_mutual_realm.gno
*/
// Pins the on-chain artifacts of a mutual type-decl cycle: the two named
// types keep distinct TypeIDs despite sharing one *StructType, and the T2
// object's persisted bytes, hashes and LastObjectSize are frozen. The PR's
// own filetest asserts stdout only, so nothing there fails if the shared
// base ever changes what gets written to state.
// Fails at the merge-base with `should not happen`; passes at 093c32be0.
package decltypemutual

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

var Head = T1{Next: &T2{Val: 9}, Val: 1}

func main(cur realm) {
	Head.Next.Val = 42
	println(Head.Next.Val)
	println(T2(Head).Val)
	println(Head.Next.Next == nil)
}

// Output:
// 42
// 1
// true

// Types:
// T1[gno.land/r/decltypemutual.T1]=
//     {
//         "ID": "gno.land/r/decltypemutual.T1"
//     }
// T2[gno.land/r/decltypemutual.T2]=
//     {
//         "ID": "gno.land/r/decltypemutual.T2"
//     }

// Realm:
// finalizerealm["gno.land/r/decltypemutual"]
// u[0a62c126be240fbf8097b8bb6de29e00c159b8db:6](5)=
//     @@ -10,7 +10,7 @@
//                  }
//              },
//              {
//     -            "N": "CQAAAAAAAAA=",
//     +            "N": "KgAAAAAAAAA=",
//                  "T": {
//                      "@type": "/gno.PrimitiveType",
//                      "value": "32"
//     @@ -20,7 +20,7 @@
//          "ObjectInfo": {
//              "ID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:6",
//              "LastObjectSize": "296",
//     -        "ModTime": "0",
//     +        "ModTime": "8",
//              "OwnerID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:5",
//              "RefCount": "1"
//          }
// u[0a62c126be240fbf8097b8bb6de29e00c159b8db:5](5)=
//     @@ -2,7 +2,7 @@
//          "ObjectInfo": {
//              "ID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:5",
//              "LastObjectSize": "344",
//     -        "ModTime": "0",
//     +        "ModTime": "8",
//              "OwnerID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:4",
//              "RefCount": "1"
//          },
//     @@ -13,7 +13,7 @@
//              },
//              "V": {
//                  "@type": "/gno.RefValue",
//     -            "Hash": "5d8d0e64bf8a1e9f486e2d58441349715fcf85d6",
//     +            "Hash": "7aa47f98574f35342930dd9c9a2b35fd68998dd6",
//                  "ObjectID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:6"
//              }
//          }
// u[0a62c126be240fbf8097b8bb6de29e00c159b8db:4](5)=
//     @@ -12,7 +12,7 @@
//                      "@type": "/gno.PointerValue",
//                      "Base": {
//                          "@type": "/gno.RefValue",
//     -                    "Hash": "a8a12021223e67e42ecb34c86fdaa17d3a9f060e",
//     +                    "Hash": "9673fd9732ff9896efca34e8e7f0b7c01bcf8530",
//                          "ObjectID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:5"
//                      },
//                      "Index": "0",
//     @@ -30,7 +30,7 @@
//          "ObjectInfo": {
//              "ID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:4",
//              "LastObjectSize": "425",
//     -        "ModTime": "0",
//     +        "ModTime": "8",
//              "OwnerID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:3",
//              "RefCount": "1"
//          }
// u[0a62c126be240fbf8097b8bb6de29e00c159b8db:3](5)=
//     @@ -2,7 +2,7 @@
//          "ObjectInfo": {
//              "ID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:3",
//              "LastObjectSize": "344",
//     -        "ModTime": "0",
//     +        "ModTime": "8",
//              "OwnerID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:2",
//              "RefCount": "1"
//          },
//     @@ -13,7 +13,7 @@
//              },
//              "V": {
//                  "@type": "/gno.RefValue",
//     -            "Hash": "ad361869b90dcbdd262d1fc11e1e54685807a02c",
//     +            "Hash": "2175fd558893728f8dd3295821cc1378da5cf4fe",
//                  "ObjectID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:4"
//              }
//          }
// u[0a62c126be240fbf8097b8bb6de29e00c159b8db:2](5)=
//     @@ -4,7 +4,7 @@
//              "ID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:2",
//              "IsEscaped": true,
//              "LastObjectSize": "741",
//     -        "ModTime": "0",
//     +        "ModTime": "8",
//              "RefCount": "2"
//          },
//          "Parent": null,
//     @@ -58,7 +58,7 @@
//                  },
//                  "V": {
//                      "@type": "/gno.RefValue",
//     -                "Hash": "97b6e088782f59dd82610927782eebfed5b23007",
//     +                "Hash": "1825999932e01bca6597645f6624e11a9602eb81",
//                      "ObjectID": "0a62c126be240fbf8097b8bb6de29e00c159b8db:3"
//                  }
//              },
