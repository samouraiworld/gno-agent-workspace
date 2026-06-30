Verified by:

- [x]  David
- [ ]  Ghost
- [x]  Lours
- [x]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From 22/06 to 29/06  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by a merger · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- ✅ **fix(tm2): use separate mutex on ABCI queries client** - https://github.com/gnolang/gno/pull/5431 - Villaquiranm
- ✅ **test(misc/e2e): add gnovm audit and e2e regression scripts** - https://github.com/gnolang/gno/pull/5663 - louis14448
- ✅ **refactor(gnovm): stream print/println & panic formatting through a buffered metered writer** - https://github.com/gnolang/gno/pull/5641 - omarsy
- **fix(gnovm): meter gas correctly for switch case** - https://github.com/gnolang/gno/pull/5217 - davd-gzl
- **docs: add new `r/docs/...` examples** - https://github.com/gnolang/gno/pull/5016 - davd-gzl
- fix(gnovm): respect Unicode range in 64-bit integer-to-string conversions - https://github.com/gnolang/gno/pull/5807 - omarsy

---

**🛡️ PR Waiting for review (Security)**

- ⚠️ 💥 fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl
- ✅ fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl
- ✅ 💥 fix(gnovm/debugger): add bounds checks to prevent index panics - https://github.com/gnolang/gno/pull/5202 - davd-gzl
- fix(gnovm): recover panics when having unhashable type as map key - https://github.com/gnolang/gno/pull/5821 - Villaquiranm
- fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl (Recurrent conflict /!\ gas)
- 📥 💥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- fix(gnovm): ignore blank fields in struct equality and map keys - https://github.com/gnolang/gno/pull/5784 - omarsy
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl
- fix(gnolang): allow indirect cur-call through a local func variable - https://github.com/gnolang/gno/pull/5689 - omarsy
- fix(preprocess): using iota outside constant declaration - https://github.com/gnolang/gno/pull/5822 - Villaquiranm
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy
- 📥 fix(gnovm): meter BigInt and BigDec comparison operators - https://github.com/gnolang/gno/pull/5646 - davd-gzl
- 📥 fix(gnolang): allow local type declarations in block statements - https://github.com/gnolang/gno/pull/5754 - davd-gzl
- 🆕 📥 fix(gnovm): fold -0 to +0 for float call args - https://github.com/gnolang/gno/pull/5864 - davd-gzl
- 🆕 📥 feat(gnovm): replace cockroachdb/apd -> math/big.Rat (breaking) - https://github.com/gnolang/gno/pull/5867 - Villaquiranm
- 📥 💥 fix(gnovm/stdlibs/strings): keep invalid UTF-8 bytes in Split, add tests - https://github.com/gnolang/gno/pull/5749 - davd-gzl
- 💥 feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy
- 📥 feat(bank): `TotalCoin` - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl (Recurrent conflict /!\ gas)
- feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - mvallenet (Recurrent conflict /!\ testdata)
- 💥 feat(validators): add attributes to validator event emissions - https://github.com/gnolang/gno/pull/5366 - mvallenet
- 📥 💥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl

---

**📖 PR Waiting for review (Documentation)**

- 📥 docs: add `make preview` target for the docs.gno.land frontend - https://github.com/gnolang/gno/pull/5752 - davd-gzl
- 🆕 📥 docs(gnodev): refresh gnodev guide and document missing flags - https://github.com/gnolang/gno/pull/5865 - davd-gzl

---

**📦 PR Waiting for review (Packages)**

- ⚠️ ✅ fix(example/avl): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5314 - davd-gzl (Recurrent conflict /!\ testdata)
- ✅ fix(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - davd-gzl
- ✅ 💥 feat(example/bptree): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5644 - davd-gzl
- feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl
- 📥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl

---

**🌐 PR Waiting for review (Gnoweb)**

- feat(gnoweb): expose render link on realm directory views - https://github.com/gnolang/gno/pull/5618 - AmozPay
- ✅ 💥 feat(gnoweb): differenciate render and dir view with $dir - https://github.com/gnolang/gno/pull/5622 - AmozPay
- 💥 feat(gnoweb): make heading text clickable to set URL hash - https://github.com/gnolang/gno/pull/5585 - davd-gzl

---

**🔧 PR Waiting for review (Tools)**

- feat(gnodev): add gnodev version command - https://github.com/gnolang/gno/pull/5563 - AmozPay
- feat: incremental sync from indexer - https://github.com/gnoverse/mygnoscan/pull/1 - Miguel

---

**📂 PR Waiting for review (Other)**

- feat(gno): load bank param from genesis_param.toml - https://github.com/gnolang/gno/pull/5370 - mvallenet
- 📥 💥 feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers - https://github.com/gnolang/gno/pull/5676 - davd-gzl
- 📥 feat(stdlibs): port encoding/ascii85 and encoding/pem - https://github.com/gnolang/gno/pull/5679 - davd-gzl (Recurrent conflict /!\ generated)

---

**🚧 PR In Progress — [Not approved by AI](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/README.md)**

- ⚠️ 💥 feat(GovDAO): add activity page to highlight inactive GovDAO's members - https://github.com/gnolang/gno/pull/4731 - davd-gzl
- ✅ feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - mvallenet (Recurrent conflict /!\ testdata)
- ✅ feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm (Recurrent conflict /!\ go.mod)
- fix(gnovm): typedRuntimeError for runtime errors - https://github.com/gnolang/gno/pull/5732 - Villaquiranm
- feat(examples): add subscriptions package - https://github.com/gnolang/gno/pull/4931 - mvallenet
- 💥 feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl
- 💥 feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - mvallenet
- 💥 feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl
- 💥 feat(tm2/std,gnovm): drop _filetest.gno suffix requirement - https://github.com/gnolang/gno/pull/5712 - davd-gzl
- 💥 docs: add cheat sheet page - https://github.com/gnolang/gno/pull/5551 - davd-gzl
- 🚫 fix(consensus): implement `RemovePeer` cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl

---

**🚧 PR In Progress — Draft**

- fix(gnovm): respect type identity in assignability - https://github.com/gnolang/gno/pull/5785 - omarsy
- fix(gnovm): depth-based shadowing for promoted struct fields and methods - https://github.com/gnolang/gno/pull/5820 - omarsy
- feat(stdlibs): port upstream additions Go 1.18-1.25 across 11 packages - https://github.com/gnolang/gno/pull/5753 - davd-gzl
- 💥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- 💥 feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy
- 💥 feat(gnoweb): add `:::details` collapsible block - https://github.com/gnolang/gno/pull/5593 - davd-gzl
- 💥 WIP: feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5619 - davd-gzl
- 💥 WIP feat(gnovm): add math/big stdlib (Int subset) - https://github.com/gnolang/gno/pull/5678 - davd-gzl
- 💥 feat(gnodev): auto-import the dev key into the local keybase - https://github.com/gnolang/gno/pull/5680 - davd-gzl

---

**🐛 Issues Opened:**

- GnoVM: Untyped float constant round-trip `(1.0/3.0)*3.0` evaluates to `false == 1.0` in Gno - https://github.com/gnolang/gno/issues/5862 - Villaquiranm
- GnoVM: `1 << 3000` accepted at parse time then panics with internal stacktrace on `float64` conversion - https://github.com/gnolang/gno/issues/5863 - Villaquiranm

---

**🎉 PR Merged**

- fix(gnovm): allow `fallthrough` from non-last default clause - https://github.com/gnolang/gno/pull/5682 - davd-gzl
- docs(builders): consolidate and clean up builder documentation - https://github.com/gnolang/gno/pull/5656 - davd-gzl
- test(gnovm): pin nil-map delete semantics (follow-up to #5196) - https://github.com/gnolang/gno/pull/5808 - omarsy
- docs(gnovm): simplify zero-sized pointer equality docs - https://github.com/gnolang/gno/pull/5836 - davd-gzl
- https://github.com/gnolang/gno/security/advisories/GHSA-7q6h-h9fq-967h - omarsy
- https://github.com/gnolang/gno-ghsa-288g-9j7f-gh9v/pull/1 - omarsy

---

**🖥️ Validators / Infrastructure Tools:**

samouraiworld/gnomonitoring

- 🆕 feat: report BFT margin - https://github.com/samouraiworld/gnomonitoring/pull/113 - louis14448

Merged this week:

- feat(db): migrate backend from SQLite to PostgreSQL - https://github.com/samouraiworld/gnomonitoring/pull/102 - louis14448
- fix(govdao): close unclosed `<b>` tags and escape proposal Title/Url - https://github.com/samouraiworld/gnomonitoring/pull/103 - louis14448
- fix(govdao): make govdaos primary key (chain_id, id) to stop cross-chain collisions - https://github.com/samouraiworld/gnomonitoring/pull/104 - louis14448
- fix(valoper): signing address moniker - https://github.com/samouraiworld/gnomonitoring/pull/105 - louis14448
- fix(govdao): resilient proposal enrichment - https://github.com/samouraiworld/gnomonitoring/pull/111 - louis14448

7 infra issues opened this week (#106-110, #112) - valoper/moniker robustness and GovDAO REJECTED/expired alerting - louis14448

---

**🤖 Onboarding Bot (/samouraiworld/gno-onboarding-bot)**

- Retry rate-limited Sheets requests; simplify approve message - https://github.com/samouraiworld/gno-onboarding-bot/pull/13 - D4ryl00

---

**📝 NOTE:**
