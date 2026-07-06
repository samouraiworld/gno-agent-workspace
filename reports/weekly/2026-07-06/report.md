Verified by:
- [ ]  David
- [ ]  Ghost
- [ ]  Lours
- [ ]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From 29/06 to 05/07  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by a merger · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- ✅ **fix(tm2): concurrent ABCI queries via snapshot isolation and atomic commits** - https://github.com/gnolang/gno/pull/5431 - Villaquiranm
- ✅ **test(misc/e2e): add gnovm audit and e2e regression scripts** - https://github.com/gnolang/gno/pull/5663 - louis14448
- 💥 **fix(gnovm): meter gas correctly for switch case** - https://github.com/gnolang/gno/pull/5217 - davd-gzl
- ✅ **docs: add new `r/docs/...` examples** - https://github.com/gnolang/gno/pull/5016 - davd-gzl

---

**🛡️ PR Waiting for review (Security)**

- ✅ fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl (AI: needs discussion)
- fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl (expected conflict: gas)
- 📥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl
- ⚠️ 💥 fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ⚠️ 📥 feat(bank): `TotalCoin` - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl (expected conflict: gas)
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl
- 🆕 fix(gnovm): reclaim stored key object on delete() of object-keyed map - https://github.com/gnolang/gno/pull/5882 - omarsy
- 🆕 fix(gnovm): remove duplicated escaped-hash delete in DelObject - https://github.com/gnolang/gno/pull/5899 - omarsy
- feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - mvallenet (expected conflict: testdata)
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy
- 📥 fix(gnovm): meter BigInt and BigDec comparison operators - https://github.com/gnolang/gno/pull/5646 - davd-gzl
- 📥 fix(gnolang): allow local type declarations in block statements - https://github.com/gnolang/gno/pull/5754 - davd-gzl
- 📥 fix(gnovm): fold -0 to +0 for float call args - https://github.com/gnolang/gno/pull/5864 - davd-gzl (AI: needs discussion)
- 🆕 📥 perf(gnovm): skip store probe for synthetic package paths - https://github.com/gnolang/gno/pull/5900 - omarsy
- 🆕 📥 perf(vm): lazily clone the type-check cache per transaction - https://github.com/gnolang/gno/pull/5901 - omarsy
- 💥 fix(gnolang): allow indirect cur-call through a local func variable - https://github.com/gnolang/gno/pull/5689 - omarsy
- 💥 feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy (AI: needs discussion)
- 📥 💥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl

---

**📖 PR Waiting for review (Documentation)**

- 📥 docs: add `make preview` target for the docs.gno.land frontend - https://github.com/gnolang/gno/pull/5752 - davd-gzl

---

**📦 PR Waiting for review (Packages)**

- ⚠️ ✅ fix(example/avl): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5314 - davd-gzl (expected conflict: testdata)
- ✅ fix(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - davd-gzl
- feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl
- 📥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl (AI: needs discussion)
- ✅ 💥 feat(example/bptree): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5644 - davd-gzl

---

**🌐 PR Waiting for review (Gnoweb)**

- feat(gnoweb): make heading text clickable to set URL hash - https://github.com/gnolang/gno/pull/5585 - davd-gzl
- feat(gnoweb): expose render link on realm directory views - https://github.com/gnolang/gno/pull/5618 - AmozPay
- ✅ 💥 feat(gnoweb): differenciate render and dir view with $dir - https://github.com/gnolang/gno/pull/5622 - AmozPay

---

**🔧 PR Waiting for review (Tools)**

- feat(gnodev): add gnodev version command - https://github.com/gnolang/gno/pull/5563 - AmozPay

---

**📂 PR Waiting for review (Other)**

- 📥 fix(gnovm/stdlibs/strings): keep invalid UTF-8 bytes in Split, add tests - https://github.com/gnolang/gno/pull/5749 - davd-gzl
- 🆕 📥 fix(valopers): validate auth-list members, sanitize description, reject negative min fee - https://github.com/gnolang/gno/pull/5874 - davd-gzl
- 📥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- 📥 feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers - https://github.com/gnolang/gno/pull/5676 - davd-gzl
- 📥 feat(stdlibs): port encoding/ascii85 and encoding/pem - https://github.com/gnolang/gno/pull/5679 - davd-gzl (expected conflict: generated)

---

**🚧 PR In Progress — [Not approved by AI](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/README.md)**

- ⚠️ 💥 feat(GovDAO): add activity page to highlight inactive GovDAO's members  - https://github.com/gnolang/gno/pull/4731 - davd-gzl
- ✅ 💥 fix(gnovm): typedRuntimeError for runtime errors - https://github.com/gnolang/gno/pull/5732 - Villaquiranm
- ✅ feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - mvallenet (expected conflict: testdata)
- ✅ feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm (expected conflict: go.mod)
- feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - mvallenet
- 🆕 📥 feat(gnovm): replace cockroachdb/apd -> math/big.Rat - https://github.com/gnolang/gno/pull/5867 - Villaquiranm
- docs: add cheat sheet page - https://github.com/gnolang/gno/pull/5551 - davd-gzl
- 🆕 📥 docs: rewrite gnokey into guide and reference, rename gnodev doc - https://github.com/gnolang/gno/pull/5873 - davd-gzl
- 💥 feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl
- 💥 feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl
- 💥 feat(tm2/std,gnovm): drop _filetest.gno suffix requirement - https://github.com/gnolang/gno/pull/5712 - davd-gzl
- 🚫 fix(consensus): implement `RemovePeer` cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl

---

**🚧 PR In Progress — Draft**

- fix(gnovm): respect type identity in assignability - https://github.com/gnolang/gno/pull/5785 - omarsy
- fix(gnovm): depth-based shadowing for promoted struct fields and methods - https://github.com/gnolang/gno/pull/5820 - omarsy
- 🆕 fix(gnovm): meter GC traversal of large primitive-keyed maps - https://github.com/gnolang/gno/pull/5884 - omarsy
- feat(stdlibs): port upstream additions Go 1.18-1.25 across 11 packages - https://github.com/gnolang/gno/pull/5753 - davd-gzl
- 🆕 perf(gnovm): load package file blocks lazily - https://github.com/gnolang/gno/pull/5902 - omarsy
- 💥 feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy
- 💥 feat(gnoweb): add `:::details` collapsible block - https://github.com/gnolang/gno/pull/5593 - davd-gzl
- 💥 feat(gnodev): auto-import the dev key into the local keybase - https://github.com/gnolang/gno/pull/5680 - davd-gzl
- 💥 WIP: feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5619 - davd-gzl
- 💥 WIP feat(gnovm): add math/big stdlib (Int subset) - https://github.com/gnolang/gno/pull/5678 - davd-gzl

---

**🐛 Issues Opened:**

- delete() on an array/struct-keyed realm map leaks the stored key object (storage not reclaimed) - https://github.com/gnolang/gno/issues/5881 - omarsy
- GC visit gas undercharges O(N) traversal of large primitive-keyed maps - https://github.com/gnolang/gno/issues/5883 - omarsy

---

**🎉 PR Merged**

- fix(gnovm/debugger): add bounds checks to prevent index panics - https://github.com/gnolang/gno/pull/5202 - davd-gzl
- refactor(gnovm): stream print/println & panic formatting through a buffered metered writer - https://github.com/gnolang/gno/pull/5641 - omarsy
- fix(gnovm): ignore blank fields in struct equality and map keys - https://github.com/gnolang/gno/pull/5784 - omarsy
- fix(gnovm): respect Unicode range in 64-bit integer-to-string conversions - https://github.com/gnolang/gno/pull/5807 - omarsy
- fix(gnovm): recover panics when having unhashable type as map key - https://github.com/gnolang/gno/pull/5821 - Villaquiranm
- fix(preprocess): using iota outside constant declaration - https://github.com/gnolang/gno/pull/5822 - Villaquiranm
- docs(gnodev): refresh gnodev guide and document missing flags - https://github.com/gnolang/gno/pull/5865 - davd-gzl
- fix(gnodev): reset to initial package set; loader root + slice fixes - https://github.com/gnolang/gno/pull/5889 - davd-gzl

---

**🖥️ Validators / Infrastructure Tools:**

- Feat/report bft margin - https://github.com/samouraiworld/gnomonitoring/pull/113 - louis14448

---

**📝 NOTE:**
