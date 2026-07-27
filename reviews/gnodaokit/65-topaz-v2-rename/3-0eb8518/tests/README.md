# Round 3 test artifacts

Adversarial fixtures for the round-3 findings on [PR 65](https://github.com/samouraiworld/gnodaokit/pull/65), run at head `0eb8518` with the toolchain the branch pins (`gnolang/gno@fc4052651`).

Every file asserts the desired post-fix state, so each one is red at `0eb8518` and green once the finding is fixed.

## Setup, shared by all of them

```bash
# from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08
```

## `authlens` — the DAO interface donates the caller's realm

Four realms. `victim` runs the framework's own cross-realm membership recipe against a DAO handle it did not create; `evil` implements `daokit.DAO` and crosses into `probe` under whatever realm it was handed; `asuite` asserts `probe` never sees the victim.

| file | destination |
|---|---|
| `probe_probe.gno` | `gno/r/authlens/probe/probe.gno` |
| `victim_victim.gno` | `gno/r/authlens/victim/victim.gno` |
| `evil_evil.gno` | `gno/r/authlens/evil/evil.gno` |
| `asuite_asuite_test.gno` | `gno/r/authlens/asuite/asuite_test.gno` |

Each directory also needs a `gnomod.toml` naming its module (`gno.land/r/samcrew/authlens/<dir>`, `gno = "0.9"`).

```bash
/tmp/gno-topaz test -v ./gno/r/authlens/asuite
```

At `0eb8518` the probe records `gno.land/r/samcrew/authlens/victim`.

## `config_reuse_probe_test.gno` — one Config, two DAOs

Destination `gno/p/basedao/config_reuse_probe_test.gno`.

```bash
/tmp/gno-topaz test -v -run 'TestConfigReuse' ./gno/p/basedao
```

At `0eb8518` the second DAO gets a 60% migration bar instead of 80%, and its default governance counts the first DAO's members.

## `render_probe_test.gno` — every render path, valid and invalid input

Destination `gno/p/basedao/render_probe_test.gno`. It prints one line per path rather than asserting, so it passes either way; read the output.

```bash
/tmp/gno-topaz test -v -run 'TestRenderProbeEveryPath' ./gno/p/basedao
```

At `0eb8518` four paths abort: `proposal/999`, `proposal/abc`, `proposal/-1`, `role/nosuchrole`. All four also abort at the merge-base `b8332969`.
