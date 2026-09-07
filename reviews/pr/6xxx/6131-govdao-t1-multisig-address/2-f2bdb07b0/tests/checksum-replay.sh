#!/usr/bin/env bash
# Replays gen-genesis.sh steps 4.1-7.3 for one deployment, which is everything
# feeding `verify_checksum "$GENESIS_TXS_JSONL"`. Prints the sha256 of
# work/valoper-seed.jsonl and work/genesis_txs.jsonl so both can be compared
# against that script's own CHECKSUMS_DATA heredoc.
#
# valoper-seed is the fidelity control: it does not derive from examples/, so a
# run that reproduces its locked value has replayed the pipeline faithfully and
# a genesis_txs mismatch is the tree, not the harness.
#
# Run: from a gnolang/gno clone, with gno, gnokey and gnogenesis built from it
# and first on PATH:
#
#   gh pr checkout 6131 -R gnolang/gno
#   mkdir -p /tmp/gnobin
#   # gnogenesis is its own module, so each binary needs `go build -C`,
#   # the same way gen-genesis.sh builds them.
#   go build -C gnovm/cmd/gno -o /tmp/gnobin/gno .
#   go build -C gno.land/cmd/gnokey -o /tmp/gnobin/gnokey .
#   go build -C contribs/gnogenesis -o /tmp/gnobin/gnogenesis .
#   export PATH=/tmp/gnobin:$PATH
#   base=$(git merge-base origin/master HEAD)
#   for rev in "$base" HEAD; do
#     rm -rf /tmp/t && mkdir -p /tmp/t
#     git archive "$rev" examples misc/deployments | tar -x -C /tmp/t
#     ./checksum-replay.sh pearl /tmp/t/examples /tmp/t/misc/deployments/pearl.gno.land /tmp/out-"$rev"
#   done
#
# Every chain-specific value, the deployer key, the chain id, the genesis time,
# the package filter and the validator set, is read out of the deployment's own
# gen-genesis.sh, so this harness holds no copy of any of them and cannot drift
# from the script it is replaying.
#
# Usage: ./checksum-replay.sh <pearl|sapphire> <examples-dir> <deployment-dir> <out-dir>
set -eu
DEPLOY="$1"; E="$2"; P="$3"; W="$4"
SCRIPT="$P/gen-genesis.sh"
[ -f "$SCRIPT" ] || { echo "no gen-genesis.sh under $P" >&2; exit 2; }

# Pull each value from the deployment script rather than restating it.
val() { sed -n "s/^$1=\\(.*\\)$/\\1/p" "$SCRIPT" | head -1 | sed 's/ *#.*//; s/^"//; s/"$//'; }
CHAIN_ID=$(val CHAIN_ID)
GENESIS_TIME=$(val GENESIS_TIME)
DEPLOYER_KEY=$(val DEPLOYER_KEY)
DEPLOYER_ADDR=$(val DEPLOYER_ADDR)
MN=$(val DEPLOYER_MNEMONIC)
for v in CHAIN_ID GENESIS_TIME DEPLOYER_KEY DEPLOYER_ADDR MN; do
  eval "[ -n \"\${$v:-}\" ]" || { echo "could not read $v from $SCRIPT" >&2; exit 2; }
done

# FILTERED_PACKAGES, one entry per line, comments stripped.
mapfile -t FILTERED < <(sed -n '/^FILTERED_PACKAGES=(/,/^)/p' "$SCRIPT" | sed -n 's/^[[:space:]]*\(\.\/[^[:space:]]*\).*/\1/p')
[ "${#FILTERED[@]}" -gt 0 ] || { echo "could not read FILTERED_PACKAGES from $SCRIPT" >&2; exit 2; }

rm -rf "$W"; mkdir -p "$W"
GH="$W/gnokey-home"
G="$W/genesis.json"; TXS="$W/genesis_txs.jsonl"

# 4.1/4.2
pkg_dirs=$(cd "$E" && gno tool deplist -test-dep "${FILTERED[@]}")
# 4.3
S="$W/examples"; mkdir -p "$S"
while IFS= read -r dir; do
  [ -z "$dir" ] && continue
  rel="${dir#"$E"/}"; dest="$S/$rel"; mkdir -p "$dest"
  find "$dir" -maxdepth 1 -type f -exec cp {} "$dest/" \;
  if [ -d "$dir/filetests" ]; then cp -r "$dir/filetests" "$dest/filetests"; fi
done <<<"$pkg_dirs"
# 4.4
printf '%s\n\n' "$MN" | gnokey add --recover "$DEPLOYER_KEY" --home "$GH" --insecure-password-stdin >/dev/null 2>&1
gnokey list --home "$GH" | grep -q "$DEPLOYER_ADDR" || { echo "DEPLOYER_ADDR mismatch"; exit 2; }
# 4.5/4.6/4.7
gnogenesis generate -chain-id "$CHAIN_ID" -genesis-time "$GENESIS_TIME" --output-path "$G" >/dev/null
echo "" | gnogenesis txs add packages "$S" -gno-home "$GH" -key-name "$DEPLOYER_KEY" --genesis-path "$G" --insecure-password-stdin >/dev/null 2>&1
gnogenesis txs export "$TXS" --genesis-path "$G" >/dev/null

msgrun() { # dir outfile
  local dir="$1" out="$2" meta="$1/meta.json" tx="$W/.txn.json"
  local reason ck bf gw gf an sq
  reason=$(jq -r .reason "$meta"); ck=$(jq -r .caller_key "$meta"); bf=$(jq -r .body_file "$meta")
  gw=$(jq -r .gas_wanted "$meta"); gf=$(jq -r .gas_fee "$meta"); an=$(jq -r .account_number "$meta"); sq=$(jq -r .sequence "$meta")
  gnokey maketx run --gas-wanted "$gw" --gas-fee "$gf" --chainid "$CHAIN_ID" --home "$GH" --broadcast=false --insecure-password-stdin "$ck" "$dir/$bf" >"$tx" <<<""
  echo "" | gnokey sign --tx-path "$tx" --chainid "$CHAIN_ID" --account-number "$an" --account-sequence "$sq" --home "$GH" --insecure-password-stdin "$ck" >/dev/null
  jq -c --arg r "$reason" '{tx: ., metadata: {block_height: "0"}, reason: $r}' "$tx" >>"$out"; rm -f "$tx"
}
msgcall() { # dir outfile
  local dir="$1" out="$2" meta="$1/meta.json" tx="$W/.txn.json"
  local reason ck co pp fn gw gf an sq
  reason=$(jq -r .reason "$meta"); ck=$(jq -r .caller_key "$meta"); co=$(jq -r '.caller_override // empty' "$meta")
  pp=$(jq -r .pkgpath "$meta"); fn=$(jq -r .func "$meta")
  gw=$(jq -r .gas_wanted "$meta"); gf=$(jq -r .gas_fee "$meta"); an=$(jq -r .account_number "$meta"); sq=$(jq -r .sequence "$meta")
  local aa=(); while IFS= read -r a; do aa+=(--args "$a"); done < <(jq -r '.args // [] | .[]' "$meta")
  echo "" | gnokey maketx call --pkgpath "$pp" --func "$fn" "${aa[@]}" --gas-wanted "$gw" --gas-fee "$gf" --chainid "$CHAIN_ID" --home "$GH" --broadcast=false --insecure-password-stdin "$ck" >"$tx"
  echo "" | gnokey sign --tx-path "$tx" --chainid "$CHAIN_ID" --account-number "$an" --account-sequence "$sq" --home "$GH" --insecure-password-stdin "$ck" >/dev/null
  if [ -n "$co" ]; then
    jq -c --arg c "$co" --arg r "$reason" '.msg[0].caller = $c | {tx: ., metadata: {block_height: "0"}, reason: $r}' "$tx" >>"$out"
  else
    jq -c --arg r "$reason" '{tx: ., metadata: {block_height: "0"}, reason: $r}' "$tx" >>"$out"
  fi
  rm -f "$tx"
}
# Step 5
: >"$W/bootstrap.jsonl"; msgrun "$P/transactions/base/bootstrap" "$W/bootstrap.jsonl"
jq -c 'del(.reason)' "$W/bootstrap.jsonl" >"$W/bootstrap_s.jsonl"
gnogenesis txs add sheets "$W/bootstrap_s.jsonl" --genesis-path "$G" >/dev/null
cat "$W/bootstrap_s.jsonl" >>"$TXS"
# Step 6
: >"$W/names.jsonl"; msgcall "$P/transactions/migration/names-enable" "$W/names.jsonl"
jq -c 'del(.reason)' "$W/names.jsonl" >"$W/names_s.jsonl"
gnogenesis txs add sheets "$W/names_s.jsonl" --genesis-path "$G" >/dev/null
cat "$W/names_s.jsonl" >>"$TXS"
# Step 7
CSV="$W/valoper.csv"; SEED="$W/valoper-seed.jsonl"
# INITIAL_VALSET rows are "name power address pubkey"; operators are positional.
mapfile -t VALSET < <(sed -n '/^INITIAL_VALSET=(/,/^)/p' "$SCRIPT" | sed -n 's/^[[:space:]]*"\(.*\)"[[:space:]]*$/\1/p')
# Operator entries are quoted in some builders and bare in others.
mapfile -t OPERATORS < <(sed -n '/^INITIAL_VALSET_OPERATORS=(/,/^)/p' "$SCRIPT" | sed -n 's/^[[:space:]]*"\?\(g1[0-9a-z]*\)"\?.*/\1/p')
[ "${#VALSET[@]}" -eq "${#OPERATORS[@]}" ] || { echo "valset and operator counts differ" >&2; exit 2; }
{
  echo "operator_addr,signing_pubkey,moniker,description,server_type"
  for i in "${!VALSET[@]}"; do
    read -r name _power _address pub_key <<<"${VALSET[$i]}"
    printf '%s,%s,%s,%s founding validator (%s),cloud\n' \
      "${OPERATORS[$i]}" "$pub_key" "$name" "$DEPLOY" "$name"
  done
} >"$CSV"
gnogenesis fork valoper-seed --csv "$CSV" --output "$SEED" --caller "$DEPLOYER_ADDR" >/dev/null
echo "  valoper-seed.jsonl sha256: $(sha256sum "$SEED" | cut -d' ' -f1)"
jq -c 'del(.reason)' "$SEED" >"$W/valoper_s.jsonl"
gnogenesis txs add sheets "$W/valoper_s.jsonl" --genesis-path "$G" >/dev/null
cat "$W/valoper_s.jsonl" >>"$TXS"
echo "  genesis_txs.jsonl sha256: $(sha256sum "$TXS" | cut -d' ' -f1)  ($(wc -l <"$TXS") txs)"
