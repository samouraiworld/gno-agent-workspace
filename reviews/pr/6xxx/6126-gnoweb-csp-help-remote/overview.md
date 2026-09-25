# Which RPC host the gnoweb Content Security Policy allows the browser to query
claude-opus-5-5

## TLDR

gnoweb talks to a gno.land node twice: from its own server, and from the visitor's browser when the Actions page runs a read-only function. The browser request goes to `-help-remote`, but the Content Security Policy only allowed `-remote`, so the browser blocked it whenever the two differ. The change builds the policy from `-help-remote`.

## What it is for

The Actions tab of a realm page lets a visitor fill in a function's arguments and see its result. The result comes from a `vm/qeval` query that the page's script sends straight to the node's `/abci_query` endpoint.

## How it works today

Two flags name the node. `-remote` is where gnoweb's server fetches pages from, often an internal address. `-help-remote` is the public address printed in the `gnokey` commands and used by the browser; left empty, it falls back to `-remote`.

Before the change, the `connect-src` directive of the policy named `-remote`:

| Flags | Browser fetches | Policy allows (before) | Result panel |
| --- | --- | --- | --- |
| only `-remote` set | `-remote` | `-remote` | loads |
| both set, same host | that host | that host | loads |
| both set, different hosts | `-help-remote` | `-remote` | blocked |

## What the change does

After the change, `connect-src` names `-help-remote`, with its fallback to `-remote` untouched:

| Flags | Policy allows (after) | Result panel |
| --- | --- | --- |
| only `-remote` set | `-remote` | loads |
| both set, same host | that host | loads |
| both set, different hosts | `-help-remote` | loads |

The first two rows produce the same header as before. A small wrapper, `newSecureHeadersMiddleware`, reads the address from the app config, and a new test calls that wrapper with two different hosts.

## Concepts

- **Content Security Policy (CSP)**: a response header listing which origins a page may load from or send requests to. `connect-src` covers `fetch()`; a request to a host not listed fails in the browser before it leaves.
- **Strict mode**: gnoweb sends the CSP header only when `-no-strict` is not set.
