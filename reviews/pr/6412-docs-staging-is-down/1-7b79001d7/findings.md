# Findings in posting order, from round assemble: 4 to post, 0 SKIP, 0 refuted kept out

## docs/resources/effective-gno.md:964 [gh](https://github.com/gnolang/gno/blob/7b79001d744ecd27adc684cd2ac44719326c75a1/docs/resources/effective-gno.md#L964) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: the foo20 See also link points at github.com/gnolang/gno/tree/master instead of the relative ../../examples/gno.land/r/demo/defi/foo20
Check: sed -n '964p' docs/resources/effective-gno.md at 7b79001d7; git grep -c '\.\./\.\./examples/' -- 'docs/*.md' for the relative convention the PR body states for links leaving docs/
Details: Line 964 at head reads the absolute master URL. The PR body says relative links that leave docs/ stay relative because they hold at the version being read, and docs/resources/community-packages.md, gno-packages.md and gnoland-networks.md link into examples/ and misc/ relatively; the four links this PR adds to replace staging URLs take the other form. No linter enforces either form.
Evidence: sed -n '964p' docs/resources/effective-gno.md: https://github.com/gnolang/gno/tree/master/examples/gno.land/r/demo/defi/foo20; git grep -c '../../examples/' docs: community-packages.md 15, gno-packages.md 4, gno-data-structures.md 3

## docs/users/explore-with-gnoweb.md:77 [gh](https://github.com/gnolang/gno/blob/7b79001d744ecd27adc684cd2ac44719326c75a1/docs/users/explore-with-gnoweb.md#L77) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: the r/docs/hello link points at github.com/gnolang/gno/tree/master instead of the relative ../../examples/quarantined/gno.land/r/docs/hello
Check: sed -n '77p' docs/users/explore-with-gnoweb.md at 7b79001d7; git grep -c '\.\./\.\./examples/' -- 'docs/*.md' for the relative convention the PR body states for links leaving docs/
Details: Line 77 at head reads the absolute master URL. The PR body says relative links that leave docs/ stay relative because they hold at the version being read, and docs/resources/community-packages.md, gno-packages.md and gnoland-networks.md link into examples/ and misc/ relatively; the four links this PR adds to replace staging URLs take the other form. No linter enforces either form.
Evidence: sed -n '77p' docs/users/explore-with-gnoweb.md: https://github.com/gnolang/gno/tree/master/examples/quarantined/gno.land/r/docs/hello; git grep -c '../../examples/' docs: community-packages.md 15, gno-packages.md 4, gno-data-structures.md 3

## docs/users/explore-with-gnoweb.md:125 [gh](https://github.com/gnolang/gno/blob/7b79001d744ecd27adc684cd2ac44719326c75a1/docs/users/explore-with-gnoweb.md#L125) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: the r/docs/source link points at github.com/gnolang/gno/tree/master instead of the relative ../../examples/quarantined/gno.land/r/docs/source
Check: sed -n '125p' docs/users/explore-with-gnoweb.md at 7b79001d7; git grep -c '\.\./\.\./examples/' -- 'docs/*.md' for the relative convention the PR body states for links leaving docs/
Details: Line 125 at head reads the absolute master URL. The PR body says relative links that leave docs/ stay relative because they hold at the version being read, and docs/resources/community-packages.md, gno-packages.md and gnoland-networks.md link into examples/ and misc/ relatively; the four links this PR adds to replace staging URLs take the other form. No linter enforces either form.
Evidence: sed -n '125p' docs/users/explore-with-gnoweb.md: https://github.com/gnolang/gno/tree/master/examples/quarantined/gno.land/r/docs/source; git grep -c '../../examples/' docs: community-packages.md 15, gno-packages.md 4, gno-data-structures.md 3

## docs/users/interact-with-gnokey.md:412 [gh](https://github.com/gnolang/gno/blob/7b79001d744ecd27adc684cd2ac44719326c75a1/docs/users/interact-with-gnokey.md#L412) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: the Counter realm link points at github.com/gnolang/gno/tree/master instead of the relative ../../examples/gno.land/r/demo/counter
Check: sed -n '412p' docs/users/interact-with-gnokey.md at 7b79001d7; git grep -c '\.\./\.\./examples/' -- 'docs/*.md' for the relative convention the PR body states for links leaving docs/
Details: Line 412 at head reads the absolute master URL. The PR body says relative links that leave docs/ stay relative because they hold at the version being read, and docs/resources/community-packages.md, gno-packages.md and gnoland-networks.md link into examples/ and misc/ relatively; the four links this PR adds to replace staging URLs take the other form. No linter enforces either form.
Evidence: sed -n '412p' docs/users/interact-with-gnokey.md: https://github.com/gnolang/gno/tree/master/examples/gno.land/r/demo/counter; git grep -c '../../examples/' docs: community-packages.md 15, gno-packages.md 4, gno-data-structures.md 3
