# Review: [#5080](https://github.com/gnolang/gno/pull/5080)
Event: COMMENT

## Body
Where this branch stands against master today.

Master already does the main thing this branch asked for: whether namespace ownership is enforced is now a chain parameter rather than a switch inside the names realm, and gno.land's genesis sets it. That was the ask on this thread, and it is settled.

Three things are still only here. A chain that sets nothing gets enforcement off rather than on. Packages deployed at genesis are exempt from the check. And the on/off switch is deleted from the names realm, where master instead kept it and built a pause and a DAO-controlled admin around it.

The branch was opened in January and last touched in March. Master has moved 371 commits since, and 8 of the 12 files here conflict.
