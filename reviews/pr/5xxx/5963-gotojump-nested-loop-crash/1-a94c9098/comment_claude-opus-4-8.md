# Review: PR [#5963](https://github.com/gnolang/gno/pull/5963)
Posted: https://github.com/gnolang/gno/pull/5963#pullrequestreview-4706229861
Event: APPROVE

## Body
Verified on a94c90986. Re-adding the deleted second truncation reproduces the slice bounds out of range [:-1] crash. Output matches go run byte-for-byte across for, range, and switch-clause frame crossings for 3- to 6-deep nesting.
