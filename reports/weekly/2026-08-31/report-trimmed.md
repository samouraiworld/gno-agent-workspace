Verified by:
- [ ]  David
- [ ]  Ghost
- [ ]  Lours
- [ ]  Mikecito
- [ ]  zôÖma

**Quick Intro Context:**

---

From 24/08 to 31/08  **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by a merger · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- ✅ 💥 fix(gnovm): meter gas correctly for switch case - https://github.com/gnolang/gno/pull/5217 - davd-gzl
- 📥 docs: concise AI contract review guide follow-up - https://github.com/gnolang/gno/pull/5936 - davd-gzl
- fix(gnolang): allow indirect cur-call through a local func variable - https://github.com/gnolang/gno/pull/5689 - omarsy
- ✅ 💥 feat(gnovm): source-level gas profiler ("gas pprof") - https://github.com/gnolang/gno/pull/5967 - omarsy

---

**🛡️ PR Waiting for review (Security)**

- ⚠️ fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl
- ✅ fix(tm2/consensus): stop a proposer timing out on its own proposal - https://github.com/gnolang/gno/pull/6006 - davd-gzl
- fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl (expected conflict: gas)
- 📥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ✅ 📥 feat(tm2): bounded-parallel queries, pre-filled VM type caches, snapshot-isolated simulate - https://github.com/gnolang/gno/pull/6082 - Villaquiranm
- ✅ test(misc/e2e): add gnovm audit and e2e regression scripts - https://github.com/gnolang/gno/pull/5663 - louis14448
- fix(preprocess): avoid shadowing of iota - https://github.com/gnolang/gno/pull/5981 - Villaquiranm (AI: needs discussion)
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy
- 📥 fix(gnovm): meter BigInt and BigDec comparison operators - https://github.com/gnolang/gno/pull/5646 - davd-gzl
- 📥 fix(gnolang): allow local type declarations in block statements - https://github.com/gnolang/gno/pull/5754 - davd-gzl
- 📥 fix(gnovm): strip inherited directives from transpiled output - https://github.com/gnolang/gno/pull/6081 - omarsy
- 📥 feat(gnovm): reject compiler and tooling directives in submitted packages - https://github.com/gnolang/gno/pull/6078 - omarsy
- 📥 perf(tm2/bptree): find the oldest and newest version with two seeks - https://github.com/gnolang/gno/pull/5979 - davd-gzl
- 📥 💥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl
- 📥 💥 perf(gnovm): compute map keys once, and skip the redundant TypeID prefix for concrete key types - https://github.com/gnolang/gno/pull/6020 - Villaquiranm

---

**📖 PR Waiting for review (Documentation)**

- ✅ docs: add `make preview` target for the docs.gno.land frontend - https://github.com/gnolang/gno/pull/5752 - davd-gzl
- 📥 docs: rewrite gnokey into guide and reference, rename gnodev doc - https://github.com/gnolang/gno/pull/5873 - davd-gzl
- 📥 💥 docs: complete the effective-gno roadmap topics - https://github.com/gnolang/gno/pull/6000 - davd-gzl (changes requested)

---

**📦 PR Waiting for review (Packages)**

- 📥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl (AI: needs discussion)
- 💥 feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl
- 📥 💥 refactor(examples)!: drop the placeholder argument by keeping the realm off first - https://github.com/gnolang/gno/pull/6033 - davd-gzl

---

**🌐 PR Waiting for review (Gnoweb)**

- ✅ 💥 fix(gnoweb): follow-up fixes for the package overview page - https://github.com/gnolang/gno/pull/5934 - davd-gzl
- 💥 feat(gnoweb): make heading text clickable to set URL hash - https://github.com/gnolang/gno/pull/5585 - davd-gzl

---

**📂 PR Waiting for review (Other)**

- ✅ fix(valopers): validate auth-list members, sanitize description, reject negative min fee - https://github.com/gnolang/gno/pull/5874 - davd-gzl
- 📥 fix(gnovm/stdlibs/strings): keep invalid UTF-8 bytes in Split, add tests - https://github.com/gnolang/gno/pull/5749 - davd-gzl (expected conflict: apphash)
- 📥 feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers - https://github.com/gnolang/gno/pull/5676 - davd-gzl (expected conflict: apphash)
- 📥 feat(stdlibs): port encoding/ascii85 and encoding/pem - https://github.com/gnolang/gno/pull/5679 - davd-gzl (expected conflict: apphash)
- 📥 chore(perfs): Cache type-privacy checks across commits - https://github.com/gnolang/gno/pull/5923 - Villaquiranm
- 📥 💥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy

---

**🐛 Issues Opened:**

- 🆕 Launch content: a blog post and dapp idea list, one per persona - https://github.com/gnolang/gno/issues/6102 - davd-gzl

---

**🎉 PR Merged**

---

**📚 Docs site (/gnolang/docs.gno.land)**

- fix: link out of the docs folder to the monorepo on GitHub - https://github.com/gnolang/docs.gno.land/pull/76 - davd-gzl

---

**🖥️ Validators / Infrastructure Tools:**

- 💥 Feat/report bft margin - https://github.com/samouraiworld/gnomonitoring/pull/113 - louis14448

---

**📝 NOTE:**
