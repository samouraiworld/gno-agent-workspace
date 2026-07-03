/* Run: from a gno checkout:
gh pr checkout 5882 -R gnolang/gno && git checkout e98021315
curl -fsSL -o gnovm/tests/files/zrealm_map6.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/tests/zrealm_map6.gno
go test -v -run 'TestFiles/zrealm_map6.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zrealm_map6.gno
*/

// PKGPATH: gno.land/r/test
package test

// Companion to zrealm_map5 with a struct key instead of an array: delete()
// must reclaim the stored struct key object, emitted as d[...:8](-252) in the
// finalize diff. Repointing the delete builtin's DidUpdate at the argument key
// instead of the stored key drops that d[...] line and the key object leaks.
type Key struct {
	A, B int
}

var m map[Key]int

func init() {
	m = map[Key]int{Key{1, 2}: 10, Key{3, 4}: 20}
}

func main(cur realm) {
	delete(m, Key{1, 2})
	println("len:", len(m))
}

// Output:
// len: 1

// Realm:
// finalizerealm["gno.land/r/test"]
// u[08ada09dee16d791fd406d629fe29bb0ed084a30:7](-193)=
//     @@ -1,26 +1,6 @@
//      {
//          "List": {
//              "List": [
//     -            {
//     -                "Key": {
//     -                    "T": {
//     -                        "@type": "/gno.RefType",
//     -                        "ID": "gno.land/r/test.Key"
//     -                    },
//     -                    "V": {
//     -                        "@type": "/gno.RefValue",
//     -                        "Hash": "dff81de12fa6de5e1a87f1a9345b7c7940191026",
//     -                        "ObjectID": "08ada09dee16d791fd406d629fe29bb0ed084a30:8"
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
//     @@ -46,7 +26,7 @@
//          "ObjectInfo": {
//              "ID": "08ada09dee16d791fd406d629fe29bb0ed084a30:7",
//              "LastObjectSize": "572",
//     -        "ModTime": "0",
//     +        "ModTime": "9",
//              "OwnerID": "08ada09dee16d791fd406d629fe29bb0ed084a30:3",
//              "RefCount": "1"
//          }
// u[08ada09dee16d791fd406d629fe29bb0ed084a30:3](0)=
//     @@ -2,7 +2,7 @@
//          "ObjectInfo": {
//              "ID": "08ada09dee16d791fd406d629fe29bb0ed084a30:3",
//              "LastObjectSize": "384",
//     -        "ModTime": "6",
//     +        "ModTime": "9",
//              "OwnerID": "08ada09dee16d791fd406d629fe29bb0ed084a30:2",
//              "RefCount": "1"
//          },
//     @@ -20,7 +20,7 @@
//              },
//              "V": {
//                  "@type": "/gno.RefValue",
//     -            "Hash": "600699518290c08fd94287112f802d54a8f55da4",
//     +            "Hash": "fdd11570a7e689ba76ce972aea2fabde0dbe53b9",
//                  "ObjectID": "08ada09dee16d791fd406d629fe29bb0ed084a30:7"
//              }
//          }
// u[08ada09dee16d791fd406d629fe29bb0ed084a30:2](0)=
//     @@ -46,7 +46,7 @@
//                  },
//                  "V": {
//                      "@type": "/gno.RefValue",
//     -                "Hash": "56c243e61c568b133bc30f2ab99b8d9ddea15cec",
//     +                "Hash": "962d51977810c5055f54f51b8f11a557701d61c5",
//                      "ObjectID": "08ada09dee16d791fd406d629fe29bb0ed084a30:3"
//                  }
//              },
// d[08ada09dee16d791fd406d629fe29bb0ed084a30:8](-252)
