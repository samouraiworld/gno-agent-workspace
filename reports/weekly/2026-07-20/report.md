Verified by:
- [ ]  David
- [ ]  Ghost
- [ ]  Lours
- [ ]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From 13/07 to 20/07  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by a merger · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- ✅ **fix(tm2): concurrent ABCI queries via snapshot isolation and atomic commits** - https://github.com/gnolang/gno/pull/5431 - Villaquiranm
- ✅ **test(misc/e2e): add gnovm audit and e2e regression scripts** - https://github.com/gnolang/gno/pull/5663 - louis14448
- **fix(gnolang): allow indirect cur-call through a local func variable** - https://github.com/gnolang/gno/pull/5689 - omarsy

---

**🛡️ PR Waiting for review (Security)**

- fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl (expected conflict: gas)
- 🆕 📥 fix(gnovm): pin the per-file Go version in the consensus type-check - https://github.com/gnolang/gno/pull/5978 - davd-gzl
- ⚠️ 💥 fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl
- ✅ 💥 fix(gnovm): meter gas correctly for switch case - https://github.com/gnolang/gno/pull/5217 - davd-gzl

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ⚠️ 📥 feat(bank): `TotalCoin` - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl (expected conflict: gas)
- 🆕 ✅ fix(gnovm): correct GotoJump stmt-stack truncation for goto out of nested loops - https://github.com/gnolang/gno/pull/5963 - omarsy
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl
- feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - mvallenet (expected conflict: testdata)
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy
- 📥 fix(gnolang): allow local type declarations in block statements - https://github.com/gnolang/gno/pull/5754 - davd-gzl
- 📥 fix(gnovm): fold -0 to +0 for float call args - https://github.com/gnolang/gno/pull/5864 - davd-gzl
- 🆕 📥 feat(gnovm): source-level gas profiler ("gas pprof") - https://github.com/gnolang/gno/pull/5967 - omarsy
- 💥 feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy (AI: needs discussion)
- 📥 💥 fix(gnovm): meter BigInt and BigDec comparison operators - https://github.com/gnolang/gno/pull/5646 - davd-gzl
- 📥 💥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl

---

**📖 PR Waiting for review (Documentation)**

- 🆕 ✅ docs(examples): add READMEs for p/nt packages - https://github.com/gnolang/gno/pull/5950 - davd-gzl
- 📥 docs: add `make preview` target for the docs.gno.land frontend - https://github.com/gnolang/gno/pull/5752 - davd-gzl
- 📥 docs: concise AI contract review guide follow-up - https://github.com/gnolang/gno/pull/5936 - davd-gzl

---

**📦 PR Waiting for review (Packages)**

- 📥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl (AI: needs discussion)
- 💥 feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl

---

**🌐 PR Waiting for review (Gnoweb)**

- feat(gnoweb): expose render link on realm directory views - https://github.com/gnolang/gno/pull/5618 - AmozPay
- 📥 fix(gnoweb): follow-up fixes for the package overview page - https://github.com/gnolang/gno/pull/5934 - davd-gzl
- ✅ 💥 feat(gnoweb): differenciate render and dir view with $dir - https://github.com/gnolang/gno/pull/5622 - AmozPay
- 💥 feat(gnoweb): make heading text clickable to set URL hash - https://github.com/gnolang/gno/pull/5585 - davd-gzl

---

**🔧 PR Waiting for review (Tools)**

- feat(gnodev): add gnodev version command - https://github.com/gnolang/gno/pull/5563 - AmozPay

---

**📂 PR Waiting for review (Other)**

- ✅ fix(valopers): validate auth-list members, sanitize description, reject negative min fee - https://github.com/gnolang/gno/pull/5874 - davd-gzl
- 📥 fix(gnovm/stdlibs/strings): keep invalid UTF-8 bytes in Split, add tests - https://github.com/gnolang/gno/pull/5749 - davd-gzl (expected conflict: apphash)
- 📥 feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers - https://github.com/gnolang/gno/pull/5676 - davd-gzl (expected conflict: apphash)
- 📥 feat(stdlibs): port encoding/ascii85 and encoding/pem - https://github.com/gnolang/gno/pull/5679 - davd-gzl (expected conflict: apphash)

---

**🚧 PR In Progress — [Not approved by AI](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/README.md)**

- ✅ feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm (expected conflict: go.mod)
- 📥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl
- 📥 chore(perfs): Cache type-privacy checks across commits - https://github.com/gnolang/gno/pull/5923 - Villaquiranm
- 💥 feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - mvallenet
- 📥 💥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- 📥 💥 docs: rewrite gnokey into guide and reference, rename gnodev doc - https://github.com/gnolang/gno/pull/5873 - davd-gzl

---

**🚧 PR In Progress — Draft**

- ⚠️ 💥 feat(GovDAO): add activity page to highlight inactive GovDAO's members - https://github.com/gnolang/gno/pull/4731 - davd-gzl (AI: changes requested)
- ✅ 💥 fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl (AI: needs discussion)
- fix(gnovm): meter GC traversal of large primitive-keyed maps - https://github.com/gnolang/gno/pull/5884 - omarsy
- 🆕 fix(gnovm): make tryEvalStatic's error result meaningful - https://github.com/gnolang/gno/pull/5977 - omarsy
- 🆕 fix(preprocess): avoid shadowing of iota - https://github.com/gnolang/gno/pull/5981 - Villaquiranm
- feat(stdlibs): port upstream additions Go 1.18-1.25 across 11 packages - https://github.com/gnolang/gno/pull/5753 - davd-gzl
- perf(vm): lazily clone the type-check cache per transaction - https://github.com/gnolang/gno/pull/5901 - omarsy
- 🆕 perf(gnovm): speed up DidUpdate per-write ownership hook - https://github.com/gnolang/gno/pull/5960 - omarsy (AI: changes requested)
- 🆕 perf(tm2/bptree): bound version discovery to two seeks - https://github.com/gnolang/gno/pull/5979 - davd-gzl
- docs: add cheat sheet page - https://github.com/gnolang/gno/pull/5551 - davd-gzl (AI: changes requested)
- 💥 fix(gnovm): respect type identity in assignability - https://github.com/gnolang/gno/pull/5785 - omarsy
- 💥 fix(gnovm): depth-based shadowing for promoted struct fields and methods - https://github.com/gnolang/gno/pull/5820 - omarsy
- 💥 feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl (AI: changes requested)
- 💥 feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl (AI: changes requested)
- 💥 feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy (AI: changes requested)
- 💥 feat(gnoweb): add `:::details` collapsible block - https://github.com/gnolang/gno/pull/5593 - davd-gzl
- 💥 WIP: feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5619 - davd-gzl
- 💥 WIP feat(gnovm): add math/big stdlib (Int subset) - https://github.com/gnolang/gno/pull/5678 - davd-gzl (AI: needs discussion)
- 💥 feat(gnodev): auto-import the dev key into the local keybase - https://github.com/gnolang/gno/pull/5680 - davd-gzl (AI: needs discussion)
- 💥 feat(tm2/std,gnovm): drop _filetest.gno suffix requirement - https://github.com/gnolang/gno/pull/5712 - davd-gzl (AI: changes requested)
- 🆕 💥 feat(examples): pluggable grc20 ledger storage + p/nt/hashmap (flat gas for large ledgers) - https://github.com/gnolang/gno/pull/5965 - omarsy
- 🚫 fix(consensus): implement `RemovePeer` cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl (AI: changes requested)

---

**🐛 Issues Opened:**

- GnoVM: selector through a defined pointer type (type D1 *D2) panics "should not happen" instead of matching Go - https://github.com/gnolang/gno/issues/5957 - omarsy

---

**🎉 PR Merged**

- fix(gnovm): qualify cross-package methods in interface errors - https://github.com/gnolang/gno/pull/5932 - davd-gzl
- perf(gnovm): load package file blocks lazily - https://github.com/gnolang/gno/pull/5902 - omarsy
- fix(gnovm): reclaim stored key object on delete() of object-keyed map - https://github.com/gnolang/gno/pull/5882 - omarsy
- fix(tm2): concurrent ABCI queries via snapshot isolation and atomic commits - https://github.com/gnolang/gno/pull/5431 - Villaquiranm
- feat(gnovm): replace cockroachdb/apd -> math/big.Rat - https://github.com/gnolang/gno/pull/5867 - Villaquiranm

---

**🖥️ Validators / Infrastructure Tools:**

- Feat/report bft margin - https://github.com/samouraiworld/gnomonitoring/pull/113 - louis14448

Merged:
- Perf/bound agrega fallback window - https://github.com/samouraiworld/gnomonitoring/pull/119 - louis14448
- Fix/valset membership integrity - https://github.com/samouraiworld/gnomonitoring/pull/120 - louis14448
- Feat/report hide departed validators - https://github.com/samouraiworld/gnomonitoring/pull/121 - louis14448
- Feat/incident rate normalization - https://github.com/samouraiworld/gnomonitoring/pull/122 - louis14448
- Feat/daily report revamp - https://github.com/samouraiworld/gnomonitoring/pull/123 - louis14448
- fix: label the daily report's Score/Missed/VP figures as last-24h - https://github.com/samouraiworld/gnomonitoring/pull/124 - louis14448

---

**📝 NOTE:**
