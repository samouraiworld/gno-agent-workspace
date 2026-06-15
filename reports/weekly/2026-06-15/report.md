Verified by:

- [ ]  Amoz
- [ ]  David
- [ ]  Ghost
- [ ]  Lours
- [ ]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From 08/06 to 15/06  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by core team · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict
> 🤖 Our AI review: ✅ approve · ❌ request changes · 💬 needs discussion

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- ✅ **fix(tm2): use separate mutex on ABCI queries client** - https://github.com/gnolang/gno/pull/5431 - Villaquiranm. Have some differences between simulate and tx on gas measuring, investigating but team input would be appreciated 🤖✅
- ✅ **test(misc/e2e): add gnovm audit and e2e regression scripts** - https://github.com/gnolang/gno/pull/5663 - louis14448 🤖✅
- **fix(gnovm): meter gas correctly for switch case** - https://github.com/gnolang/gno/pull/5217 - davd-gzl 🤖✅
- **docs: add new `r/docs/...` examples** - https://github.com/gnolang/gno/pull/5016 - davd-gzl 🤖✅
- ✅ 💥 **feat(example/bptree): simplify `Get` to return `nil` as "no value"** - https://github.com/gnolang/gno/pull/5644 - davd-gzl 🤖✅
- 💥 **refactor(gnovm): stream Protected*String through allocWriter for per-byte gas accounting** - https://github.com/gnolang/gno/pull/5641 - omarsy (In progress)
- **[UX-9] Developer journey data & builder insights** - https://github.com/gnolang/gno/issues/5467 - Miguel : timeseries (render charts) to check on-chain sanity, PR coming soon. *(carried from last week, status to refresh)*

---

**🛡️ PR Waiting for review (Security)**

- ⚠️ 📥 fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl 🤖✅
- ✅ fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl 🤖💬
- fix(gnovm): avoid having empty pointer if range nil array is discarded - https://github.com/gnolang/gno/pull/5733 - Villaquiranm 🤖✅
- fix(gnovm): avoid panic on assertion over nil slice - https://github.com/gnolang/gno/pull/5780 - Villaquiranm 🤖✅
- 🆕 fix(gnovm): recover panics when having unhashable type as map key - https://github.com/gnolang/gno/pull/5821 - Villaquiranm 🤖✅
- 🚫 fix(consensus): implement `RemovePeer` cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl 🤖❌
- ⚠️ ✅ 💥 fix(example/avl): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5314 - davd-gzl 🤖✅
- ✅ 💥 fix(gnovm/debugger): add bounds checks to prevent index panics - https://github.com/gnolang/gno/pull/5202 - davd-gzl 🤖✅
- 💥 fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl 🤖✅
- 📥 💥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl 🤖✅

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ✅ fix(gnovm): allow `fallthrough` from non-last default clause - https://github.com/gnolang/gno/pull/5682 - davd-gzl 🤖✅
- ✅ revert(validators): remove valset trust-level and cooldown limits - https://github.com/gnolang/gno/pull/5767 - omarsy 🤖✅
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl 🤖✅
- fix(gnolang): allow indirect cur-call through a local func variable - https://github.com/gnolang/gno/pull/5689 - omarsy 🤖✅
- fix(gnovm): ignore blank fields in struct equality and map keys - https://github.com/gnolang/gno/pull/5784 - omarsy 🤖✅
- 🆕 fix(gnovm): respect Unicode range in 64-bit integer-to-string conversions - https://github.com/gnolang/gno/pull/5807 - omarsy 🤖✅
- 🆕 fix(gnovm): correct softfloat add/sub for normal operands cancelling to subnormal - https://github.com/gnolang/gno/pull/5818 - omarsy 🤖✅
- 🆕 fix(preprocess): using iota outside constant declaration - https://github.com/gnolang/gno/pull/5822 - Villaquiranm 🤖✅
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl 🤖✅
- 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy 🤖✅
- 📥 fix(gnovm): meter BigInt and BigDec comparison operators - https://github.com/gnolang/gno/pull/5646 - davd-gzl 🤖✅
- 📥 fix(gnovm): typedRuntimeError for runtime errors - https://github.com/gnolang/gno/pull/5732 - Villaquiranm 🤖❌
- 📥 fix(gnolang): allow local type declarations in block statements - https://github.com/gnolang/gno/pull/5754 - davd-gzl 🤖✅
- 🆕 📥 test(gnovm): pin nil-map delete semantics (follow-up to #5196) - https://github.com/gnolang/gno/pull/5808 - omarsy 🤖✅
- ✅ 💥 feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - mvallenet 🤖❌
- 💥 feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - mvallenet 🤖❌
- 💥 feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy 🤖💬
- 💥 feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - mvallenet 🤖✅
- 💥 feat(validators): add attributes to validator event emissions - https://github.com/gnolang/gno/pull/5366 - mvallenet 🤖✅
- 📥 💥 fix(gnovm/stdlibs/strings): keep invalid UTF-8 bytes in Split, add tests - https://github.com/gnolang/gno/pull/5749 - davd-gzl 🤖✅
- 📥 💥 feat(bank): `TotalCoin` - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl 🤖✅
- 📥 💥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl 🤖✅

---

**📖 PR Waiting for review (Documentation)**

- ✅ docs(builders): consolidate and clean up builder documentation - https://github.com/gnolang/gno/pull/5656 - davd-gzl 🤖💬
- 📥 docs: add `make preview` target for the docs.gno.land frontend - https://github.com/gnolang/gno/pull/5752 - davd-gzl 🤖✅

---

**📦 PR Waiting for review (Packages)**

- ✅ fix(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - davd-gzl 🤖✅
- feat(examples): add subscriptions package - https://github.com/gnolang/gno/pull/4931 - mvallenet 🤖❌
- 📥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl 🤖💬
- 💥 feat(GovDAO): add activity page to highlight inactive GovDAO's members - https://github.com/gnolang/gno/pull/4731 - davd-gzl 🤖❌
- 💥 feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl 🤖❌
- 💥 feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl 🤖✅

---

**🌐 PR Waiting for review (Gnoweb)**

- feat(gnoweb): make heading text clickable to set URL hash - https://github.com/gnolang/gno/pull/5585 - davd-gzl 🤖✅
- feat(gnoweb): expose render link on realm directory views - https://github.com/gnolang/gno/pull/5618 - AmozPay 🤖✅
- ✅ 💥 feat(gnoweb): differenciate render and dir view with $dir - https://github.com/gnolang/gno/pull/5622 - AmozPay 🤖✅

---

**🔧 PR Waiting for review (Tools)**

- feat(gnokms): add insecure flag - https://github.com/gnolang/gno/pull/5360 - mvallenet 🤖❌
- feat(gnodev): add gnodev version command - https://github.com/gnolang/gno/pull/5563 - AmozPay 🤖✅
- 📥 feat(gnokey): print pkgpath after `maketx addpkg` - https://github.com/gnolang/gno/pull/5608 - davd-gzl 🤖✅
- feat(gnoscan): incremental sync from indexer - https://github.com/gnoverse/mygnoscan/pull/1 - Miguel *(carried, verify)*

---

**📂 PR Waiting for review (Other)**

- feat(gno): load bank param from genesis_param.toml - https://github.com/gnolang/gno/pull/5370 - mvallenet 🤖✅
- ✅ 💥 feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm 🤖❌
- 💥 feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl 🤖❌
- 📥 💥 feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers - https://github.com/gnolang/gno/pull/5676 - davd-gzl 🤖✅
- 📥 💥 feat(stdlibs): port encoding/ascii85 and encoding/pem - https://github.com/gnolang/gno/pull/5679 - davd-gzl 🤖✅

---

**🚧 PR In Progress:**

- fix(gnovm): respect type identity in assignability - https://github.com/gnolang/gno/pull/5785 - omarsy
- 🆕 fix(gnovm): depth-based shadowing for promoted struct fields and methods - https://github.com/gnolang/gno/pull/5820 - omarsy
- feat(gnodev): auto-import the dev key into the local keybase - https://github.com/gnolang/gno/pull/5680 - davd-gzl
- feat(stdlibs): port upstream additions Go 1.18-1.25 across 11 packages - https://github.com/gnolang/gno/pull/5753 - davd-gzl
- 💥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- 💥 feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy 🤖❌
- 💥 feat(gnoweb): add `:::details` collapsible block - https://github.com/gnolang/gno/pull/5593 - davd-gzl 🤖✅
- 💥 feat(tm2/std,gnovm): drop _filetest.gno suffix requirement - https://github.com/gnolang/gno/pull/5712 - davd-gzl 🤖❌
- 💥 docs: add cheat sheet page - https://github.com/gnolang/gno/pull/5551 - davd-gzl 🤖❌
- 💥 WIP: feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5619 - davd-gzl
- 💥 WIP feat(gnovm): add math/big stdlib (Int subset) - https://github.com/gnolang/gno/pull/5678 - davd-gzl 🤖💬

---

**🐛 Issues Opened:**

- gnovm: interface type identity does not flatten embedded interface method sets - https://github.com/gnolang/gno/issues/5810 - omarsy
- gnovm: runtime value type identity is TypeID-based (ignores struct tags, embedded syntax, variadicity) - https://github.com/gnolang/gno/issues/5817 - omarsy
- GnoVM preprocess wrongly rejects valid promoted fields/methods that the type-checker accepts - https://github.com/gnolang/gno/issues/5819 - omarsy

---

**🎉 PR Merged**

- feat(gnovm): display storage usage after running file tests - https://github.com/gnolang/gno/pull/5350 - davd-gzl
- feat(gnovm): add `errors.Unwrap`, `errors.Is`, and `errors.Join` to stdlib - https://github.com/gnolang/gno/pull/5385 - davd-gzl
- fix(examples/urequire): delegate `NotAborts` to `uassert.NotAborts` - https://github.com/gnolang/gno/pull/5672 - davd-gzl
- fix(p/nt/markdown/sanitize): reject any query in mailto URLs - https://github.com/gnolang/gno/pull/5743 - davd-gzl
- fix: nil ptr when discarded key and value not present on range - https://github.com/gnolang/gno/pull/5751 - Villaquiranm
- feat(grc721): require token owner for SetTokenMetadata - https://github.com/gnolang/gno/pull/5792 - davd-gzl

---

**🖥️ Validators / Infrastructure Tools:**

_samouraiworld/gnomonitoring_

- 🆕 feat(db): migrate backend from SQLite to PostgreSQL - https://github.com/samouraiworld/gnomonitoring/pull/102 - louis14448
- 🎉 Merged: feat(gnovalidator): add peer-based moniker discovery as fallback source - https://github.com/samouraiworld/gnomonitoring/pull/100 - louis14448
- 🎉 Merged: fix(api): run CORS before Clerk auth so OPTIONS preflight succeeds - https://github.com/samouraiworld/gnomonitoring/pull/101 - louis14448
- 🐛 Issue: feat(valoper): enrich moniker map via RPC /status with node-ID verification - https://github.com/samouraiworld/gnomonitoring/issues/98 - louis14448
- 🐛 Issue: feat(report): add BFT margin and proposer count to daily Discord/Telegram report - https://github.com/samouraiworld/gnomonitoring/issues/99 - louis14448

---

**📝 NOTE:**
