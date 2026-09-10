# Claims in comment_claude-opus-5.md, round 2 at 02ac71476

Every falsifiable claim, the check that would prove it false, and what the check printed. Reads are `git show 02ac71476:<path>`; runs use `gno` built from `./gnovm/cmd/gno` at that sha with the pinned Go, in a worktree returned clean after each.

| # | Claim | Check | Observed | Holds |
| --- | --- | --- | --- | --- |
| C1 | Body: `init()` runs once, when the realm is deployed | probe filetest prints before any call | `boards after init: 1` on the first line of every probe | yes |
| C2 | Body: `gnoland1` carries `boards2/v1` and an `OpenDiscussions` board at ID 1 | `abci_query vm/qrender gno.land/r/gnoland/boards2/v1:` on rpc.gno.land | `[OpenDiscussions] ... #1`, created 2026-03-21, then `atomone-governance` | yes |
| C3 | Body, as posted: the `init()` lands on mainnet at launch | `docs/CONSTITUTION.md:133`, `docs/resources/gnoland-networks.md` | mainnet is "the day $GNOT becomes transferrable" on the running chain; no fresh genesis named | no; removed from the posted review by hand, and from the draft |
| C4 | `boards.gno:69` is the `addBoard(..., "OpenDiscussions", true, true)` call | show lines 68-70 | line 69 as claimed | yes |
| C5 | `CreateRepost` sets a caller-chosen title and body under `PermissionThreadRepost` at `public.gno:340` | show lines 330-350 | `WithPermission(caller, PermissionThreadRepost, ...)` then `repost.Title = title`, `repost.Body = ...` | yes |
| C6 | no validator on `PermissionThreadRepost` | `git grep PermissionThreadRepost` over the package | 8 grant sites, no `ValidateFunc` | yes |
| C7 | `RequiredAccountAmount` is 3,000 GNOT at `boards.gno:25` | show line 25 | `int64(3_000_000_000)` | yes |
| C8 | `permissions.gno:111-115` makes three permissions public | show | `SetPublicPermissions(ThreadCreate, ThreadRepost, ReplyCreate)` | yes |
| C9 | `permissions.gno:157-158` attach validators to two of them | show | `ValidateFunc(PermissionThreadCreate, ...)`, `ValidateFunc(PermissionReplyCreate, ...)` | yes |
| C10 | repro: unfunded caller reposts, threads 1 to 2 | block extracted from the draft, run | `threads before: 1`, `repost id: 2`, `threads after: 2`, `gas: 3821479, storage: ...+15104b` | yes |
| C11 | same file with `CreateThread` refuses | swapped call, run | `caller is not allowed to create threads: account amount is lower than 3000 GNOT` | yes |
| C12 | `z_create_thread_06_filetest.gno` pins that message | `git grep` | that file and `z_create_reply_15` only | yes |
| C13 | removal falls to the multisig, the only `RoleOwner` | show `permissions.gno:153`, `public.gno:398-408` | `SetUserRoles(owner, RoleOwner)`; delete for owner or creator, else `PermissionThreadDelete` | yes |
| C14 | `render.gno:131-135` renders "Currently there are no boards" on an empty listed index | show | as claimed | yes |
| C15 | nothing removes from `gListedBoardsByID`; one `Set` at `public.gno:176` | `git grep` | 6 sites: declaration, `Set`, four reads | yes |
| C16 | `z_ui_home_02` line 1 still says "when there are no boards" while its golden lists a board | show lines 1, 15-19 | header unchanged, golden shows `OpenDiscussions` | yes |
| C17 | the empty page was asserted by three filetests at base and none at head | `git grep -l` at 0828bf0b3 and 02ac71476 | 3, then 0 | yes |
| C18 | `validateOpenThreadCreate` at `:111-127` exempts owners and admins only | show | `HasRole(RoleOwner) \|\| HasRole(RoleAdmin)` then `checkAccountHasAmount` | yes |
| C19 | repro: 100 GNOT refused | block extracted, run | the message above, `gas: 3452935` | yes |
| C20 | 3,000 GNOT posts `thread 1` | block with the balance changed, run | `thread 1`, PASS | yes |
| C21 | setting the threshold in `init()` reddens exactly two goldens | `RequiredAccountAmount = int64(1_000_000)` as first line, whole suite | FAIL `z_create_reply_15`, `z_create_thread_06`, nothing else | yes |
| C22 | `addBoard` at `public.gno:157` overwrites every field but the ID | show 157-185 | Name, Creator, Meta, Listed, Permissions, Threads all assigned | yes |
| C23 | `storage.Add` at `storage.gno:99-112` is a set whose only error is a nil board | show | as claimed | yes |
| C24 | both callers draw the ID from `gBoardsSequence.Next()` | `git grep` | `boards.gno:69`, `public.gno:143` into `:149` | yes |
| C25 | the patch drops `boards.New` from both call sites and the suite stays green | `git apply tests/addboard-id-param.patch`, `gno lint`, suite | one `boards.New(id)` left at `public.gno:157`; lint silent; `ok 30.38s` | yes |
| C26 | all four anchors sit in the diff | `post-pr-review.py --dry-run` | 4 comments accepted | yes |
