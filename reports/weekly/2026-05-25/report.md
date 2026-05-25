Verified by:
- [ ]  Amoz
- [ ]  David
- [ ]  Ghost
- [ ]  Lours
- [ ]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From 18/05 to 25/05  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by core team · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- 🆕 ✅ **test(misc/e2e): add gnovm audit and e2e regression scripts** - https://github.com/gnolang/gno/pull/5663 - louis14448
- **fix(gnovm): meter gas correctly for switch case** - https://github.com/gnolang/gno/pull/5217 - davd-gzl
- 🆕 💥 **feat(example/bptree): simplify `Get` to return `nil` as "no value"** - https://github.com/gnolang/gno/pull/5644 - davd-gzl
- 💥 **docs: add new `r/docs/...` examples** - https://github.com/gnolang/gno/pull/5016 - davd-gzl
- **docs: add getting started (alternative to #5519)** - https://github.com/gnolang/gno/pull/5592 - davd-gzl (merged)
- **feat(validators): limit valset changes** - https://github.com/gnolang/gno/pull/4834 - omarsy (merged)
- 🆕 **fix(gnolang): O(N^2) in Go2Gno Span for BinaryExpr chains** - https://github.com/gnolang/gno/pull/5648 - omarsy (In progress)
- 🆕 **refactor(gnovm): stream Protected*String through allocWriter for per-byte gas accounting** - https://github.com/gnolang/gno/pull/5641 - omarsy (In progress)
- **fix(tm2): use separate mutex on ABCI queries client** - https://github.com/gnolang/gno/pull/5431 - Villaquiranm (Have some differences between simulate and tx on gas measuring, investigating but team input would be appreciated)

---

**🛡️ PR Waiting for review (Security)**

- ✅ fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change - https://github.com/gnolang/gno/pull/5198 - MikaelVallenet
- ✅ 🚫 fix(consensus): handle conflicting votes instead of panicking - https://github.com/gnolang/gno/pull/5216 - davd-gzl
- fix(gnovm): add truncation protection to ProtectedString for slices, arrays, and maps - https://github.com/gnolang/gno/pull/5155 - davd-gzl
- fix(gnovm/debugger): add bounds checks to prevent index panics - https://github.com/gnolang/gno/pull/5202 - davd-gzl
- fix: prevent path traversal in `pkgdownload.Download` and `MemPackage.WriteTo` - https://github.com/gnolang/gno/pull/5219 - davd-gzl
- fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl
- feat(gno): load bank param from genesis_param.toml - https://github.com/gnolang/gno/pull/5370 - MikaelVallenet
- 🚫 fix(consensus): implement `RemovePeer` cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl
- ⚠️ 💥 fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl
- ✅ 💥 fix(gnovm): Add panic on `Deepfill` execution on constant type - https://github.com/gnolang/gno/pull/4891 - davd-gzl
- 💥 fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl
- 📥 💥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ✅ feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - MikaelVallenet
- ✅ feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm (Waiting on core team decision, RPC vs WebSocket. See also #4950)
- fix(gnovm): allow []byte -> string cast on realm owned fields - https://github.com/gnolang/gno/pull/4831 - Villaquiranm
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl
- 🆕 fix(gnoland): recover validator changes after node restart - https://github.com/gnolang/gno/pull/5469 - omarsy
- 🆕 fix(gnovm): allow `fallthrough` from non-last default clause - https://github.com/gnolang/gno/pull/5682 - davd-gzl
- 🆕 fix(gnolang): allow indirect cur-call through a local func variable - https://github.com/gnolang/gno/pull/5689 - omarsy
- feat(gnovm): consume gas when we preprocess - https://github.com/gnolang/gno/pull/4571 - omarsy
- feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - MikaelVallenet
- feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy
- feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - MikaelVallenet
- feat(validators): add attributes to validator event emissions - https://github.com/gnolang/gno/pull/5366 - MikaelVallenet
- feat(gnovm): add `errors.Unwrap`, `errors.Is`, and `errors.Join` to stdlib - https://github.com/gnolang/gno/pull/5385 - davd-gzl
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 🆕 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy
- 🆕 📥 fix(gnovm): meter BigInt and BigDec comparison operators - https://github.com/gnolang/gno/pull/5646 - davd-gzl
- 📥 💥 feat(bank): `TotalCoin` - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl
- 📥 💥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl
- 💥 feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl
- 💥 feat(gnovm): display storage usage after running file tests - https://github.com/gnolang/gno/pull/5350 - davd-gzl

---

**📖 PR Waiting for review (Documentation)**

- 🆕 docs: add editor setup guide - https://github.com/gnolang/gno/pull/5553 - davd-gzl
- 🆕 📥 docs: list per-function stdlib gaps in compatibility doc - https://github.com/gnolang/gno/pull/5677 - davd-gzl
- 🆕 📥 💥 docs(builders): consolidate and clean up builder documentation - https://github.com/gnolang/gno/pull/5656 - davd-gzl
- 🆕 💥 docs: add cheat sheet page - https://github.com/gnolang/gno/pull/5551 - davd-gzl

---

**📦 PR Waiting for review (Packages)**

- ⚠️ fix(example/avl): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5314 - davd-gzl
- ✅ fix(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - davd-gzl
- feat(examples): add subscriptions package - https://github.com/gnolang/gno/pull/4931 - MikaelVallenet
- 🆕 feat(examples/urequire): add missing uassert wrappers - https://github.com/gnolang/gno/pull/5673 - davd-gzl
- 📥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl
- ✅ 💥 feat(daokit): update daokit framework with latest version - https://github.com/gnolang/gno/pull/4884 - davd-gzl
- 🆕 💥 fix(examples/urequire): delegate `NotAborts` to `uassert.NotAborts` - https://github.com/gnolang/gno/pull/5672 - davd-gzl
- 💥 feat(GovDAO): add activity page to highlight inactive GovDAO's members - https://github.com/gnolang/gno/pull/4731 - davd-gzl
- 💥 feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl
- 💥 feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl
- 📥 💥 feat(govdao): upgrade UI/UX - https://github.com/gnolang/gno/pull/5051 - davd-gzl

---

**🌐 PR Waiting for review (Gnoweb)**

- 🆕 feat(gnoweb): make heading text clickable to set URL hash - https://github.com/gnolang/gno/pull/5585 - davd-gzl
- 🆕 feat(gnoweb): expose render link on realm directory views - https://github.com/gnolang/gno/pull/5618 - AmozPay
- 🆕 feat(gnoweb): differenciate render and dir view with $dir - https://github.com/gnolang/gno/pull/5622 - AmozPay

---

**🔧 PR Waiting for review (Tools)**

- feat(gnokms): add insecure flag - https://github.com/gnolang/gno/pull/5360 - MikaelVallenet
- 🆕 📥 feat(gnokey): print pkgpath after `maketx addpkg` - https://github.com/gnolang/gno/pull/5608 - davd-gzl

---

**📂 PR Waiting for review (Other)**

- feat: bech32 address from public key - https://github.com/gnolang/gno/pull/4506 - Villaquiranm
- 🆕 feat(gnodev): add gnodev version command - https://github.com/gnolang/gno/pull/5563 - AmozPay
- 🆕 📥 💥 feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers - https://github.com/gnolang/gno/pull/5676 - davd-gzl
- 🆕 📥 💥 feat(stdlibs): port encoding/ascii85 and encoding/pem - https://github.com/gnolang/gno/pull/5679 - davd-gzl

---

**🚧 PR In Progress:**

- feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy
- 🆕 feat(gnoweb): add `:::details` collapsible block - https://github.com/gnolang/gno/pull/5593 - davd-gzl
- 🆕 feat(gnodev): auto-import the dev key into the local keybase - https://github.com/gnolang/gno/pull/5680 - davd-gzl
- 🆕 feat(tm2/std,gnovm): drop _filetest.gno suffix requirement - https://github.com/gnolang/gno/pull/5712 - davd-gzl
- 🆕 💥 WIP: feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5619 - davd-gzl
- 🆕 💥 WIP feat(gnovm): add math/big stdlib (Int subset) - https://github.com/gnolang/gno/pull/5678 - davd-gzl

---

**🐛 Issues Opened:**

- 🐞 gnovm: unexpected unreal object panic on cross-realm call with aliased foreign object - https://github.com/gnolang/gno/issues/5694 - omarsy

---

**🎉 PR Merged**

- feat(validators): limit valset changes - https://github.com/gnolang/gno/pull/4834 - omarsy
- fix: consume gas on ComputeMapKey - https://github.com/gnolang/gno/pull/5127 - Villaquiranm
- docs: add getting started (alternative to #5519) - https://github.com/gnolang/gno/pull/5592 - davd-gzl
- feat(gnoweb): accept `gno.land` URLs in search bar - https://github.com/gnolang/gno/pull/5612 - davd-gzl

---

**🖥️ Validators / Infrastructure Tools:**

---

**📝 NOTE:**
