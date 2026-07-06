/* Run: from a gno checkout:
gh pr checkout 5882 -R gnolang/gno && git checkout e98021315
curl -fsSL -o gnovm/tests/files/zrealm_map6.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/tests/zrealm_map6.gno
go test -v -run 'TestFiles/zrealm_map6.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zrealm_map6.gno
*/

// PKGPATH: gno.land/r/test
package test

// Over-reclaim guard: the array key value is also held through keep, so the map
// holds its own copy. delete() reclaims exactly the map's copy, emitted as
// d[...:9](-213), and keep stays readable. Repointing the delete builtin's
// DidUpdate at the argument key drops that d[...] line.
var m map[[1]int]int
var keep *[1]int

func init() {
	k := [1]int{1}
	m = map[[1]int]int{k: 10, [1]int{2}: 20}
	keep = &k
}

func main(cur realm) {
	delete(m, [1]int{1})
	println("len:", len(m), "keep:", keep[0])
}

// Output:
// len: 1 keep: 1

// Realm:
// finalizerealm["gno.land/r/test"]
// u[08ada09dee16d791fd406d629fe29bb0ed084a30:8](-202)=
//     @@ -1,31 +1,6 @@
//      {
//          "List": {
//              "List": [
//     -            {
//     -                "Key": {
//     -                    "T": {
//     -                        "@type": "/gno.ArrayType",
//     -                        "Elt": {
//     -                            "@type": "/gno.PrimitiveType",
//     -                            "value": "32"
//     -                        },
//     -                        "Len": "1",
//     -                        "Vrd": false
//     -                    },
//     -                    "V": {
//     -                        "@type": "/gno.RefValue",
//     -                        "Hash": "8717c5e34966ab9eb407bea8dfe18bb110808738",
//     -                        "ObjectID": "08ada09dee16d791fd406d629fe29bb0ed084a30:9"
//     -                    }
//     -                },
//     -                "Value": {
//     -                    "N": "CgAAAAAAAAA=",
//     -                    "T": {
//     -                        "@type": "/gno.PrimitiveType",
//     -                        "value": "32"
//     -                    }
//     -                }
//     -            },
//                  {
//                      "Key": {
//                          "T": {
//     @@ -56,7 +31,7 @@
//          "ObjectInfo": {
//              "ID": "08ada09dee16d791fd406d629fe29bb0ed084a30:8",
//              "LastObjectSize": "591",
//     -        "ModTime": "0",
//     +        "ModTime": "12",
//              "OwnerID": "08ada09dee16d791fd406d629fe29bb0ed084a30:3",
//              "RefCount": "1"
//          }
// u[08ada09dee16d791fd406d629fe29bb0ed084a30:3](0)=
//     @@ -2,7 +2,7 @@
//          "ObjectInfo": {
//              "ID": "08ada09dee16d791fd406d629fe29bb0ed084a30:3",
//              "LastObjectSize": "393",
//     -        "ModTime": "7",
//     +        "ModTime": "12",
//              "OwnerID": "08ada09dee16d791fd406d629fe29bb0ed084a30:2",
//              "RefCount": "1"
//          },
//     @@ -25,7 +25,7 @@
//              },
//              "V": {
//                  "@type": "/gno.RefValue",
//     -            "Hash": "d9326ed7b65c4b60a65c827a7d11c40383191c56",
//     +            "Hash": "c51972ffb1aee4538d6efd11bca218a7a05197cc",
//                  "ObjectID": "08ada09dee16d791fd406d629fe29bb0ed084a30:8"
//              }
//          }
// u[08ada09dee16d791fd406d629fe29bb0ed084a30:2](0)=
//     @@ -34,7 +34,7 @@
//                  },
//                  "V": {
//                      "@type": "/gno.RefValue",
//     -                "Hash": "c80703c1f683e89085cb4548878fa9a985e85eec",
//     +                "Hash": "856189ab775a5091e8cec1690cb2a9aa9e900325",
//                      "ObjectID": "08ada09dee16d791fd406d629fe29bb0ed084a30:3"
//                  }
//              },
// d[08ada09dee16d791fd406d629fe29bb0ed084a30:9](-213)
