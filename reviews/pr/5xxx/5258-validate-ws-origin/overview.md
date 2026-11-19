# fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins`

Written by claude-opus-5-5.

PR: [gnolang/gno#5258](https://github.com/gnolang/gno/pull/5258)

## TLDR

Any website a user opened could connect to the node running on that user's machine and call its RPC methods, because the node accepted every live connection whatever page started it. The node now accepts such a connection only from the websites its existing cross-site setting already lists, and gnodev's live-reload connection only from gnodev's own page. A page on an allowed site and a command-line client work as before; any other site is turned away.

## The `Origin` header on a WebSocket handshake

A browser opens a WebSocket with an ordinary HTTP request that asks to upgrade the connection. On that request the browser always adds an `Origin` header naming the site of the page that ran the script, and a script cannot change or remove it. Unlike a cross-site `fetch`, a WebSocket is not covered by the browser's CORS rules: the browser lets the page read whatever the server sends back. So the server is the only place where a connection from a foreign site can be refused, and `Origin` is what it decides on. A program outside a browser, `curl` or a Go client, usually sends no `Origin` at all, and since a browser always sends one, a request without it cannot come from a web page.

## `CheckOrigin` in gorilla/websocket

Both servers in this change use `gorilla/websocket`, whose `Upgrader.CheckOrigin` decides whether an upgrade goes ahead. Before the change both set it to a function returning `true`, which accepts every site. When `CheckOrigin` is `nil`, gorilla uses its own same-origin check: a request with no `Origin` passes, and otherwise the host part of `Origin` must equal the request's `Host` header, compared as strings. To that check `localhost:8888` and `127.0.0.1:8888` are two different hosts, and so are a public name and the internal address a reverse proxy may put in `Host`.

## `cors_allowed_origins` and `OriginAllowed`

The node's `cors_allowed_origins` setting (`RPCConfig.CORSAllowedOrigins`) already decided which sites may call the HTTP RPC from a browser. Its default is `["*"]`, any site; `[]` turns CORS off; a list such as `["https://gno.land", "https://*.gno.land"]` names the allowed sites, where `*` matches any subdomain. The change reuses that list for the WebSocket: [`NewWebsocketManager`](https://github.com/gnolang/gno/pull/5258/files#diff-424c6fdbace51795f33d280588e82f2eb8292bd108799f92c06edc6d9904c4b2R1133-R1161) takes it as `allowedOrigins`, builds an `rs/cors` validator from it, and wraps the validator's `OriginAllowed` method in its `CheckOrigin`. `OriginAllowed` reads the request's `Origin` and answers whether it is on the list; a list containing `"*"` accepts everything.

The wrapper accepts a request with no `Origin` before asking the validator. Such a request comes from a program rather than a page, and no foreign site can make a browser send one, so refusing it protects nobody and would cut off every command-line and Go client the moment an operator narrows the list. The HTTP side behaves the same way: `rs/cors` passes a request without `Origin` through untouched, and gorilla's own check accepts it too.

An empty list is kept away from `rs/cors` on purpose, because `rs/cors` given no origins allows them all. The constructor builds the validator only for a non-empty list and otherwise leaves `CheckOrigin` `nil`, which is gorilla's same-origin check above.

The constructor after the change, at the pinned head:

```go
func NewWebsocketManager(funcMap map[string]*RPCFunc, allowedOrigins []string, wsConnOptions ...func(*wsConnection)) *WebsocketManager {
	// nil defaults to gorilla/websocket's same-origin check.
	var checkOrigin func(r *http.Request) bool
	if len(allowedOrigins) > 0 {
		corsValidator := cors.New(cors.Options{
			AllowedOrigins: allowedOrigins,
		})
		checkOrigin = func(r *http.Request) bool {
			return r.Header.Get("Origin") == "" || corsValidator.OriginAllowed(r)
		}
	}
```

## How the node passes the list

[`startRPC` in `node.go`](https://github.com/gnolang/gno/pull/5258/files#diff-83913b6c6e92b6851951c0f50d813df7becd09b5cd235dabfcbc584be15941f8R950-R951) hands `NewWebsocketManager` the configured list as it is, an empty one included, and the constructor's length check above turns `[]` into the same-origin check. The same list wraps the HTTP mux in `rs/cors` a few lines below, but only when `IsCorsEnabled()` is true, which is `len(CORSAllowedOrigins) != 0`. What `/websocket` answers after the change, per setting and per kind of request, from the package's own `TestWebsocketManagerCheckOrigin` and the node-level `TestNodeRPCWebsocketOrigin`:

| `cors_allowed_origins` | Page on a listed site | Page on another site | No `Origin` (a CLI or Go client) |
| --- | --- | --- | --- |
| `["*"]`, the default | accepted | accepted | accepted |
| `[]` | accepted when its host string equals `Host` | refused | accepted |
| `["https://gno.land", "https://*.gno.land"]` | accepted | refused | accepted |

A client that sends an `Origin` naming the node's own address is accepted under `[]`, since that is the same-origin case, and refused under a narrowed list that does not name the address. That refusal is what stops a page whose domain was re-pointed at the node's address, since such a page also sends an `Origin` equal to `Host`. A proxy that rewrites `Host` breaks the same-origin case under `[]`, and the operator then lists the public origin instead.

## gnodev's hot-reload socket

gnodev injects a small script into every page it serves, which opens a WebSocket to `/_events` and reloads the page when a file changes, a transaction lands or the node resets. The [emitter server](https://github.com/gnolang/gno/pull/5258/files#diff-3a40d04a36a53b11df30c032efff77d4151aa3266c66aadd43a0d13bb8d1abc8R28-R31) sets `CheckOrigin: nil`, so that connection goes through gorilla's same-origin check, which compares the page's host with the host the socket was opened on.

The [script](https://github.com/gnolang/gno/pull/5258/files#diff-e4a27fe10af055ca8ed71ae5836ece67b49156522ff896b6ceb5bce49ef7ea77R8) therefore builds its address from the page itself, `window.location.host + '/_events'` behind `wss://` on an `https` page and `ws://` otherwise, and the middleware is handed only the path, through [`emitter.NewMiddleware("/_events", webhandler)`](https://github.com/gnolang/gno/pull/5258/files#diff-45c151c403f8d498b5ef47f79408d925d7430302f0e21d141694421fd91fcfd1R493) in `app.go`. A page opened at `http://localhost:8888` then dials `localhost:8888`, and one opened at `http://127.0.0.1:8888` dials `127.0.0.1:8888`, so both pass the string comparison. A fixed `-web-listener` address in the script would pass only when the user typed exactly that address.

## Before and after

**Before:** any page on any site could open `/websocket` on a node, and `/_events` on gnodev, and the node then ran whatever RPC calls arrived on that connection, through `WebsocketManager.WebsocketHandler`.

**After:** `/websocket` follows `cors_allowed_origins` through `OriginAllowed`, or the same-origin check when the list is empty, and accepts a request with no `Origin` under every setting; `/_events` follows the same-origin check, and gnodev's script dials the host its page came from.

## Words used here

| Name | What it is |
| --- | --- |
| `Origin` | Request header a browser adds on every WebSocket handshake, naming the scheme, host and port of the page; a page cannot change it. |
| `Upgrader.CheckOrigin` | gorilla/websocket hook that returns whether an upgrade request may proceed; `nil` means the built-in same-origin check. |
| same-origin check | gorilla's default: passes a request without `Origin`, else requires `Origin`'s host to equal the `Host` header as a string. |
| `cors_allowed_origins` | Node setting `RPCConfig.CORSAllowedOrigins`, default `["*"]`; `[]` disables CORS on HTTP and leaves `/websocket` same-origin; entries may use one `*` for a subdomain. |
| `IsCorsEnabled` | `RPCConfig` method, true when `cors_allowed_origins` is not empty; decides whether the HTTP mux is wrapped in `rs/cors`. |
| `OriginAllowed` | `rs/cors` method that reads `Origin` from a request and reports whether the configured list admits it. |
| `NewWebsocketManager` | Constructor of the node's `/websocket` handler; takes `allowedOrigins` as its second argument. |
| `NewMiddleware` | gnodev emitter function that injects the hot-reload script into HTML pages; takes the socket's path, `/_events`. |
| `/_events` | gnodev's hot-reload WebSocket path, served by the emitter's `Server`. |

<details>
<summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test.

### 1. `hot` · The node's origin check, [`tm2/pkg/bft/rpc/lib/server/handlers.go`](https://github.com/gnolang/gno/pull/5258/files#diff-424c6fdbace51795f33d280588e82f2eb8292bd108799f92c06edc6d9904c4b2R1133-R1161)

Builds `CheckOrigin` from `allowedOrigins`, accepting a request with no `Origin`, or leaves gorilla's same-origin check for an empty list.

**If wrong:** a foreign site drives the node's RPC over a WebSocket, or a command-line client is turned away.

### 2. `warm` · Where the list comes from, [`tm2/pkg/bft/node/node.go`](https://github.com/gnolang/gno/pull/5258/files#diff-83913b6c6e92b6851951c0f50d813df7becd09b5cd235dabfcbc584be15941f8R950-R951)

Passes `cors_allowed_origins` to the constructor as configured.

**If wrong:** the setting stops governing `/websocket`.

### 3. `hot` · gnodev's hot-reload socket, [`contribs/gnodev/pkg/emitter/server.go`](https://github.com/gnolang/gno/pull/5258/files#diff-3a40d04a36a53b11df30c032efff77d4151aa3266c66aadd43a0d13bb8d1abc8R28-R31)

Replaces the always-true check with gorilla's same-origin default.

**If wrong:** a foreign page listens to gnodev's events, or gnodev's own page stops reloading.

### 4. `warm` · The script's address, [`contribs/gnodev/pkg/emitter/static/hotreload.js`](https://github.com/gnolang/gno/pull/5258/files#diff-e4a27fe10af055ca8ed71ae5836ece67b49156522ff896b6ceb5bce49ef7ea77R8)

Dials `/_events` on the host that served the page.

**If wrong:** a page opened under another name for the same address stops reloading.

### 5. `warm` · The usage example, [`tm2/pkg/bft/rpc/lib/doc.go`](https://github.com/gnolang/gno/pull/5258/files#diff-f106f0867319f4bd37f3615ef41382a47066a8f7042f96d64b1d612c69542f83R69)

Shows the new constructor argument with `[]string{"*"}`.

**If wrong:** a reader copies a call that no longer compiles.

### 6. `cold` · The origin table test, [`tm2/pkg/bft/rpc/lib/server/handlers_test.go`](https://github.com/gnolang/gno/pull/5258/files#diff-73126bc9535b042767665d603cda1e2d4884add7750fc5296523413dfdbff23fR978-R1046)

`TestWebsocketManagerCheckOrigin` runs one upgrade per pair of allowed list and `Origin`, a narrowed list with no `Origin` included.

**If wrong:** a regression in the check ships green.

</details>
