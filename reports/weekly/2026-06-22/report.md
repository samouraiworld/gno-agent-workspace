Verified by:

- [ ]  Amoz
- [ ]  David
- [ ]  Ghost
- [ ]  Lours
- [ ]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From 15/06 to 22/06  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by core team · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- ✅ **fix(tm2): use separate mutex on ABCI queries client** - https://github.com/gnolang/gno/pull/5431 - Villaquiranm. Fix some feedback from Morgan
- ✅ **test(misc/e2e): add gnovm audit and e2e regression scripts** - https://github.com/gnolang/gno/pull/5663 - louis14448
- ✅ 💥 **docs(builders): consolidate and clean up builder documentation** - https://github.com/gnolang/gno/pull/5656 - davd-gzl
- **fix(gnovm): meter gas correctly for switch case** - https://github.com/gnolang/gno/pull/5217 - davd-gzl
- **docs: add new `r/docs/...` examples** - https://github.com/gnolang/gno/pull/5016 - davd-gzl
- 💥 **refactor(gnovm): stream print/println & panic formatting through a buffered metered writer** - https://github.com/gnolang/gno/pull/5641 - omarsy (In progress)
- **[UX-9] Developer journey data & builder insights** - https://github.com/gnolang/gno/issues/5467 - Miguel : added a new comment and demo about the charts we should show on builder insights. Please take a look
- fix(gnovm): ignore blank fields in struct equality and map keys - https://github.com/gnolang/gno/pull/5784 - omarsy
- fix(gnovm): respect Unicode range in 64-bit integer-to-string conversions - https://github.com/gnolang/gno/pull/5807 - omarsy
- https://github.com/gnolang/gno/security/advisories/GHSA-288g-9j7f-gh9v - omarsy
- https://github.com/gnolang/gno/security/advisories/GHSA-7q6h-h9fq-967h - omarsy

---

**🛡️ PR Waiting for review (Security)**

- ⚠️ 📥 fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl
- ✅ fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl
- ✅ 💥 fix(gnovm/debugger): add bounds checks to prevent index panics - https://github.com/gnolang/gno/pull/5202 - davd-gzl
- 🆕 fix(gnovm): recover panics when having unhashable type as map key - https://github.com/gnolang/gno/pull/5821 - Villaquiranm
- 💥 fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl
- 📥 💥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ✅ fix(gnovm): allow `fallthrough` from non-last default clause - https://github.com/gnolang/gno/pull/5682 - davd-gzl
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl
- fix(gnolang): allow indirect cur-call through a local func variable - https://github.com/gnolang/gno/pull/5689 - omarsy
- fix(preprocess): using iota outside constant declaration - https://github.com/gnolang/gno/pull/5822 - Villaquiranm
- 💥 feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy
- 📥 fix(gnovm): meter BigInt and BigDec comparison operators - https://github.com/gnolang/gno/pull/5646 - davd-gzl
- 📥 test(gnovm): pin nil-map delete semantics (follow-up to #5196) - https://github.com/gnolang/gno/pull/5808 - omarsy
- 💥 feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - mvallenet
- 💥 feat(validators): add attributes to validator event emissions - https://github.com/gnolang/gno/pull/5366 - mvallenet
- 📥 💥 feat(bank): `TotalCoin` - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl
- 📥 💥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl
- 📥 💥 fix(gnovm/stdlibs/strings): keep invalid UTF-8 bytes in Split, add tests - https://github.com/gnolang/gno/pull/5749 - davd-gzl
- 📥 💥 fix(gnolang): allow local type declarations in block statements - https://github.com/gnolang/gno/pull/5754 - davd-gzl

---

**📖 PR Waiting for review (Documentation)**

- 🆕 📥 docs(gnovm): simplify zero-sized pointer equality docs - https://github.com/gnolang/gno/pull/5836 - davd-gzl
- 📥 docs: add `make preview` target for the docs.gno.land frontend - https://github.com/gnolang/gno/pull/5752 - davd-gzl

---

**📦 PR Waiting for review (Packages)**

- ⚠️ ✅ 💥 fix(example/avl): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5314 - davd-gzl
- ✅ fix(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - davd-gzl
- ✅ 💥 feat(example/bptree): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5644 - davd-gzl
- 📥 💥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl
- 💥 feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl

---

**🌐 PR Waiting for review (Gnoweb)**

- feat(gnoweb): make heading text clickable to set URL hash - https://github.com/gnolang/gno/pull/5585 - davd-gzl
- feat(gnoweb): expose render link on realm directory views - https://github.com/gnolang/gno/pull/5618 - AmozPay
- ✅ 💥 feat(gnoweb): differenciate render and dir view with $dir - https://github.com/gnolang/gno/pull/5622 - AmozPay

---

**🔧 PR Waiting for review (Tools)**

- feat(gnodev): add gnodev version command - https://github.com/gnolang/gno/pull/5563 - AmozPay
- feat(gnoscan): incremental sync from indexer - https://github.com/gnoverse/mygnoscan/pull/1 - Miguel *(carried from last week, refresh)*

---

**📂 PR Waiting for review (Other)**

- feat(gno): load bank param from genesis_param.toml - https://github.com/gnolang/gno/pull/5370 - mvallenet
- 📥 💥 feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers - https://github.com/gnolang/gno/pull/5676 - davd-gzl
- 📥 💥 feat(stdlibs): port encoding/ascii85 and encoding/pem - https://github.com/gnolang/gno/pull/5679 - davd-gzl

---

**🚧 PR In Progress — [Not approved by AI](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/README.md)**

- ⚠️ 💥 feat(GovDAO): add activity page to highlight inactive GovDAO's members - https://github.com/gnolang/gno/pull/4731 - davd-gzl
- ✅ 💥 feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - mvallenet
- ✅ 💥 feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm
- 🚫 fix(consensus): implement `RemovePeer` cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl
- 💥 feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl
- 💥 feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - mvallenet
- 💥 feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl
- 💥 feat(tm2/std,gnovm): drop _filetest.gno suffix requirement - https://github.com/gnolang/gno/pull/5712 - davd-gzl
- 💥 docs: add cheat sheet page - https://github.com/gnolang/gno/pull/5551 - davd-gzl
- fix(gnovm): typedRuntimeError for runtime errors - https://github.com/gnolang/gno/pull/5732 - Villaquiranm
- feat(examples): add subscriptions package - https://github.com/gnolang/gno/pull/4931 - mvallenet
- feat(gnokms): add insecure flag - https://github.com/gnolang/gno/pull/5360 - mvallenet

---

**🚧 PR In Progress — Draft**

- 💥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- 💥 feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy
- 💥 feat(gnoweb): add `:::details` collapsible block - https://github.com/gnolang/gno/pull/5593 - davd-gzl
- 💥 WIP: feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5619 - davd-gzl
- 💥 WIP feat(gnovm): add math/big stdlib (Int subset) - https://github.com/gnolang/gno/pull/5678 - davd-gzl
- fix(gnovm): respect type identity in assignability - https://github.com/gnolang/gno/pull/5785 - omarsy
- fix(gnovm): depth-based shadowing for promoted struct fields and methods - https://github.com/gnolang/gno/pull/5820 - omarsy
- feat(gnodev): auto-import the dev key into the local keybase - https://github.com/gnolang/gno/pull/5680 - davd-gzl
- feat(stdlibs): port upstream additions Go 1.18-1.25 across 11 packages - https://github.com/gnolang/gno/pull/5753 - davd-gzl

---

**🐛 Issues Opened:**

- *(none opened this week)*

---

**🎉 PR Merged**

- feat(gnokey): print pkgpath after `maketx addpkg` - https://github.com/gnolang/gno/pull/5608 - davd-gzl
- fix(gnovm): avoid having empty pointer if range nil array is discarded - https://github.com/gnolang/gno/pull/5733 - Villaquiranm
- revert(validators): remove valset trust-level and cooldown limits - https://github.com/gnolang/gno/pull/5767 - omarsy
- fix(gnovm): avoid panic on assertion over nil slice - https://github.com/gnolang/gno/pull/5780 - Villaquiranm
- fix(gnovm): correct softfloat add/sub for normal operands cancelling to subnormal - https://github.com/gnolang/gno/pull/5818 - omarsy
- docs: scope AI-agent disclosure to autonomous work only - https://github.com/gnolang/gno/pull/5831 - davd-gzl

---

**🖥️ Validators / Infrastructure Tools:**

samouraiworld/gnomonitoring

- 🆕 feat(db): migrate backend from SQLite to PostgreSQL - https://github.com/samouraiworld/gnomonitoring/pull/102 - louis14448

---

**🤖 Onboarding Bot (/samouraiworld/gno-onboarding-bot)**

- Auto-fill /submit-request from r/gnops/valopers - https://github.com/samouraiworld/gno-onboarding-bot/pull/4 - davd-gzl
- Feat/ghcr image publish - https://github.com/samouraiworld/gno-onboarding-bot/pull/3 - louis14448
- Harden reviewer commands and submission handling - https://github.com/samouraiworld/gno-onboarding-bot/pull/2 - louis14448
- feat: end-of-window competency digest harvest - https://github.com/samouraiworld/gno-onboarding-bot/pull/1 - davd-gzl

---

**📝 NOTE:**
