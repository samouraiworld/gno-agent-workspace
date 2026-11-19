# `whatsnew.sh`: the What's new block of the static home page

Written by claude-opus-5-5, high effort, overview agent.
PR: [gnolang/gno#6434](https://github.com/gnolang/gno/pull/6434)

## TLDR

The gno.land home page is a fixed file, so nothing on it changes between two
deployments. A new script, run every few minutes, writes the latest X and blog
posts and this week's most used realms and packages into marked places of that
file. Visitors get a home page that shows the project moving, and the testnet
gets a page of its own for builders, with the week's deployments.

## How the home page is served today

**Before:** gnoweb serves the home from a markdown file given to its
`--aliases` flag, and [`gnoalias.sh`](https://github.com/gnolang/gno/blob/d0cbe58c3/misc/deployments/home-alias/gnoalias.sh#L26-L40)
can only swap that file for `home-override.md` or append `extra-blocks.md`.
Whatever the file says at deployment is what every visitor reads until the
next one.

## What the What's new block is

On mainnet, two places of [`home.mainnet.md`](https://github.com/gnolang/gno/pull/6434/files#diff-1de9b4cc307b9739b84c96a85afc2147d951c3f81f607b33597fd91306617be1R103-R104)
fill themselves: a What's new section with a row of three X posts and a row of
three blog posts, and, inside Explore packages and realms, three lists of
[top realms, top packages and releases](https://github.com/gnolang/gno/pull/6434/files#diff-1de9b4cc307b9739b84c96a85afc2147d951c3f81f607b33597fd91306617be1R128-R129).
On a testnet, [`home.testnet.md`](https://github.com/gnolang/gno/pull/6434/files#diff-fc76144ecfd0d0f602cc87b5207f6b1901718b59b1c70eb26da30a2d74f04f75R53-R54)
becomes an onboarding page and its one block, written by `--builders`, shows
the week's totals, recent deployments, top realms and merged pull requests.

## How it works, in 5 steps

1. A timer runs `whatsnew.sh home.md`. [`page_ok`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR509-R515)
   refuses the page with exit 2 unless its marker lines are each present once,
   in order.
2. Each source whose interval has passed runs in the background,
   through [`refresh`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR329-R343):
   X, the blog, releases, merged pull requests, imports and, with an indexer,
   realm usage.
3. A source prints one tab-separated row per item into its file under
   `STATE_DIR`, the text already escaped. After: an X post row, as `row`
   prints it for a sample post on the head:
   `20261118	https://x.com/_gnoland/status/1	New release\: gno v1\.7 is out \*today\*`
4. [`render_block`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR396-R406)
   and [`render_trending`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR418-R432)
   turn those files into `<gno-columns>` markdown; that row becomes a card
   headed `**[Gno.land \@\_gnoland · Nov 18](https://x.com/...)**`.
5. [`main`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR589-R609)
   writes the page with the blocks between their markers into a temporary
   file, swaps it in only when it differs, then runs `ON_CHANGE`.

## Marker lines

The block lives between `<!-- whats-new:start -->` and
`<!-- whats-new:end -->`, and the Explore lists between the two
`whats-trending` lines. Markdown renders an HTML comment as nothing, so a page
deployed without the script shows its static content and no hole.
[`fill_block`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR518-R523)
replaces every line between a pair, which is why `page_ok` insists on one of
each and on the two pairs never nesting: a stray end marker would make the fill
eat the rest of the page.

## Sources and the state folder

Each source is a `source_*` function run as a child of the script, and its
last good output stays in `STATE_DIR`, by default `.whatsnew` beside the page.
`refresh` touches `<source>.tried` before the run, so a failing source waits
its interval like a working one: the timer can fire every 10 minutes while the
indexer is scanned once a day, `USED_TTL` 1440, and imports once a week,
`IMPORTED_TTL` 10080. A failure keeps the previous file. A ranking or
deployment list older than a week is dropped by
[`is_recent`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR354-R356),
since it would no longer read as this week.

## Escaping outside text

The page is trusted operator markdown, so any text from X, the blog or GitHub
could otherwise add a link, an image or a `<gno-*>` tag.
[`to_md`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR103-R113)
flattens the text to one line, cuts it at 110 characters, backslashes every
ASCII punctuation mark and writes the `g` of a bare `g1` address as `&#103;`.
After: one sample run through `to_md` on the head.

```text
in:  Read [the post](https://evil.example) <b>now</b> @alice, g1pq6p8lx0kk3h2kq8m3rq9v2fzy3j4w6c7a8b9d!
out: Read \[the post\]\(https\:\/\/evil\.example\) \<b\>now\<\/b\> \@alice\, &#103;1pq6p8lx0kk3h2kq8m3rq9v2fzy3j4w6c7a8b9d\!
```

Every link on the block is built by the script from a checked identifier.
For X, [`expand_links`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR81-R98)
also drops each URL of a post, keeping only a bare domain as plain text.

## Text from X

FxTwitter, a third-party mirror, only supplies the post IDs.
[`source_x`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR122-R155)
reads each post's text, date and author from X's own oEmbed endpoint and
skips a post whose author is not `X_ACCOUNT`. A third party can choose which of
the account's posts appear, never what they say.

## Ranking realms by people

[`source_used`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR241-R251)
counts distinct calling addresses per realm over the last week, since bots
dominate raw call counts. Only realms under a namespace of `USED_NAMESPACES`,
by default `gnoland gnoswap akkadia`, can appear, since any other path and any
number of addresses are free to create. To find the week,
[`scan_week`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR209-R228)
estimates the height of a week ago from the block rate over the last 20000
blocks, then queries the indexer in bands of 25000 heights, halving a band the
indexer refuses down to 250. The same arithmetic as gnoweb's chain map,
copied into the script.

## Live deployments on a testnet

On onyx, `addpkg` stores a package that stays offline until it is approved.
[`live_rows`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR258-R270)
lists a deployment only once `vm/qfile` can read it, keeps one row per
namespace and name, and checks at most 20 deployments, so one busy project
cannot crowd out the rest.

## Writing the page

The new page is written to a temporary file in the same folder with the same
mode, so the final `mv` is an atomic rename gnoweb never reads half of. An
unchanged page is left alone, and so is a page whose block would come out
empty, so a host that lost its state during an outage keeps its last filled
block. `ON_CHANGE` is a command run after a change, for a gnoweb that reads
static pages at startup only; a failed one leaves `reload-pending` and is
retried on the next run.

## The parts, at a glance

| Part | File | Job |
| --- | --- | --- |
| `whatsnew.sh` | [`misc/deployments/home-alias/whatsnew.sh`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR1-R618) | Reads the sources, renders, writes the page |
| Offline tests | [`misc/deployments/home-alias/whatsnew_test.sh`](https://github.com/gnolang/gno/pull/6434/files#diff-72df4d4f1e85bc94d17ff66d75fcd0216b011a3f366e243d3bed0cc21111ed75R1-R240) | Runs the script with curl, the RPC and the indexer stubbed |
| Mainnet home | [`home.mainnet.md`](https://github.com/gnolang/gno/pull/6434/files#diff-1de9b4cc307b9739b84c96a85afc2147d951c3f81f607b33597fd91306617be1R103-R129) | Gains both marker pairs, loses the three static Explore columns |
| Testnet home | [`home.testnet.md`](https://github.com/gnolang/gno/pull/6434/files#diff-fc76144ecfd0d0f602cc87b5207f6b1901718b59b1c70eb26da30a2d74f04f75R1-R113) | Rewritten for builders, with a deploy command and one marker pair |
| README and ADR | `misc/deployments/home-alias/README.md`, `gno.land/adr/pr6434_home_whats_new.md` | Variables, a systemd timer, the reasons behind each choice |

## Upgrading

The served page now differs from the committed one by its blocks. A
deployment restores the committed file, and the next run fills it again from
`STATE_DIR`, which must sit outside any folder a deployment replaces. Each page
takes its own `STATE_DIR`, and the script is pointed at the file itself, never
a symlink.

## Words used here

| Word | Meaning |
| --- | --- |
| `STATE_DIR` | Folder holding each source's last good output and its `.tried` stamp; default `.whatsnew` beside the page. |
| `FEED_TTL`, `USED_TTL`, `IMPORTED_TTL` | Minutes between two reads of a source: 30, 1440 and 10080 by default. |
| `USED_NAMESPACES` | Namespaces whose realms may appear in Top realms; empty allows all, as on a testnet. |
| `ON_CHANGE` | Shell command run after the page changed; empty by default. |
| `--builders` | Mode for a testnet page: totals, deployments, top realms and merged pull requests instead of posts. |
| `--dry-run` | Prints the blocks and leaves the page untouched. |
| `--force` | Reads every source now, ignoring the intervals. |
| FxTwitter | Third-party API listing an X account's posts; used for post IDs only. |
| oEmbed | X's public endpoint returning a post's HTML, author and date. |
| tx-indexer | GraphQL service over the chain's transactions, at `INDEXER_URL`; without it, Top realms and deployments stay out. |
| `vm/qfile` | RPC query reading a package's files; it answers only once the package is live. |

<details>
<summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a
generated file.

### 1. `hot` · The run and the page write, [`misc/deployments/home-alias/whatsnew.sh`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR536-R610)
Decides which sources run, and replaces the page only when its new content is
non-empty and different.
**If wrong:** a half-written or emptied home page reaches every visitor.
```bash
  tmp=$(mktemp "$home.XXXXXX")
  PAGE_TMP=$tmp
  trap 'rm -f -- "$PAGE_TMP"' EXIT
  cp -p "$home" "$tmp"
```

### 2. `hot` · Marker checks and the fill, [`misc/deployments/home-alias/whatsnew.sh`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR500-R523)
Accepts only one start and one end per pair, in order, and replaces what sits
between them.
**If wrong:** the fill swallows the static part of the page.
```bash
fill_block() {
  START="$1" END="$2" BLOCK="$3" awk '
    $0 == ENVIRON["START"] { print; printf "%s", ENVIRON["BLOCK"]; skip = 1; next }
    $0 == ENVIRON["END"] { skip = 0 }
    !skip'
}
```

### 3. `hot` · Escaping, [`misc/deployments/home-alias/whatsnew.sh`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR103-R113)
Turns any outside text into inert markdown on one line.
**If wrong:** a post puts a link or a tag of its choice on the home page.
```bash
    | gsub("(?<c>[!-/:-@\\[-`{-~])"; "\\\(.c)")
    | gsub("g(?=1[0-9A-Za-z]{36})"; "&#103;")'
```

### 4. `hot` · Refresh and the state folder, [`misc/deployments/home-alias/whatsnew.sh`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR329-R343)
Runs a source at most once per interval and keeps its last good output on
failure.
**If wrong:** a source outage blanks its row, or a slow source is hit on every
timer tick.
```bash
  touch "$stamp"
  tmp=$(mktemp "$fragment.XXXXXX")
  if "$BASH" "$0" --source "$1" >"$tmp" 2>"$tmp.err" && [ -s "$tmp" ]; then
    mv "$tmp" "$fragment"
```

### 5. `hot` · Week scan and ranking, [`misc/deployments/home-alias/whatsnew.sh`](https://github.com/gnolang/gno/pull/6434/files#diff-b99657985bc8fff6f80a24862dd3927a0728f735f8430131849db62dcf4c7b3bR209-R270)
Finds the week's heights, counts distinct callers per allowed realm, and keeps
only live deployments.
**If wrong:** Top realms shows a partial week, or a realm anyone can game.
```bash
  [ -z "$USED_NAMESPACES" ] || realms="^gno.land/r/(${USED_NAMESPACES// /|})/"
```

### 6. `hot` · The testnet page, [`misc/deployments/home-alias/home.testnet.md`](https://github.com/gnolang/gno/pull/6434/files#diff-fc76144ecfd0d0f602cc87b5207f6b1901718b59b1c70eb26da30a2d74f04f75R1-R113)
Replaces the copy of the mainnet home with onboarding and a deploy command.
**If wrong:** a builder copies an `addpkg` command that fails on onyx.

### 7. `hot` · Offline tests, [`misc/deployments/home-alias/whatsnew_test.sh`](https://github.com/gnolang/gno/pull/6434/files#diff-72df4d4f1e85bc94d17ff66d75fcd0216b011a3f366e243d3bed0cc21111ed75R1-R240)
Stubs curl, the RPC and the indexer, and checks the script's output.
**If wrong:** a regression in the script goes out unnoticed.

### 8. `cold` · The mainnet page, [`misc/deployments/home-alias/home.mainnet.md`](https://github.com/gnolang/gno/pull/6434/files#diff-1de9b4cc307b9739b84c96a85afc2147d951c3f81f607b33597fd91306617be1R103-R129)
Places both marker pairs and shortens the Explore section around them.
**If wrong:** the block lands in the wrong section of the mainnet home.

</details>
