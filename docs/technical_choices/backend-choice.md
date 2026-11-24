# Choosing Go for the AREA Backend

We are a team of four. Our backend must provide a shared API for the web and Android clients, detect events even when the clients are closed, and run the associated workflows. The brief requires at least five services and ten Actions and REActions combined. At least half of the services must require OAuth2 authorization. See the AREA brief, sections “Hooks”, “Clients & Server”, and “Deliverables and expectations”.

## Prototype Comparison

We compared Go, Rust, and TypeScript in three spikes limited to 25 minutes each. We used the same target for all three. The server had to read its port from `PORT`, be able to listen on `8080`, and return exactly `{"status":"ok"}` for `GET /ping`.

| Technology | Prototype HTTP dependencies | Time to first success | Most time-consuming step |
| --- | --- | --- | --- |
| Go | `net/http` from the standard library | 12 min | Looking up how to proceed in the Go documentation. |
| Rust | Axum and Tokio | 20 min | Figuring out how to set up the server. |
| TypeScript | Express on Node.js | 5 min | Refreshing our knowledge of the language. |

All three prototypes reached the target. The times cover setting up the route and configuring the port during each spike. Everyone on the team already knew TypeScript, which made it more predictable to get started with. [Spike comparison](../spikes/DISTANCE.md) · [Go code](../spikes/Go/server.go) · [Rust code](../spikes/Rust/src/main.rs) · [TypeScript code](../spikes/Typescript/main.ts)

### Reproduce the Go Prototype

From the repository root, start the server with port `8080` available.

```sh
cd docs/spikes/Go
PORT=8080 go run .
```

In a second terminal, `curl -fsS http://localhost:8080/ping` should return `{"status":"ok"}`.

## Why Go Fits AREA

- **A shared API for both clients.** We can build routes with `net/http`, including the route that exposes the catalogue of Services, Actions, and REActions. With `httptest`, we can send requests to handlers and check their responses. These tools give us a starting point for the test required by the brief, which must fail if the implementation departs from the OpenAPI contract, including its error responses. [net/http](https://pkg.go.dev/net/http) · [httptest](https://pkg.go.dev/net/http/httptest)
- **Autonomous hooks.** We can use goroutines to detect events alongside the HTTP server and `context` to shut down those tasks cleanly. The engine controls the order of reactions within each workflow. We also need to decide how to identify and track events we have already processed, as the brief requires. [Goroutines](https://go.dev/doc/effective_go#goroutines) · [context](https://pkg.go.dev/context)
- **Calls to third-party services.** The `net/http` client lets us set timeouts for outbound requests. If a service called by a reaction takes too long to respond, we can report the failure to the engine. We can share this component across integrations while keeping each service’s settings and permissions separate. [Go HTTP client](https://pkg.go.dev/net/http#Client)
- **An explicit workflow model.** Go structs let us represent triggers, reactions, and conditions with typed data. We can validate the rules from the brief when saving a workflow. For example, we can check that it has at most one condition and that every reference points to data declared by an earlier node.
- **A backend we can integrate with Docker Compose.** `go build` produces a server executable. We can compile it in a container build stage and run it from the final image. This gives us a starting point for the reproducible build required by the brief. [go build command](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)

## Security and Dependencies

The Go prototype uses only the standard library for this route. The Rust and TypeScript prototypes use external HTTP dependencies. For all three options, OAuth2 security depends on the library we choose, how it is maintained, and how we use it. We plan to integrate OAuth2 for the third-party services required by the brief. The library choice is still open.

For AREA, we need to verify OAuth2 callback validation, limit the permissions we request, and protect the tokens the server stores for hooks. A first experiment could connect a real service and check that the backend rejects an invalid OAuth2 callback. The current spike covers server startup and the behavior of `/ping`. [OAuth 2.0 security recommendations](https://www.rfc-editor.org/rfc/rfc9700.html)

## Decision

We chose **Go** because we wanted to learn a new technology as a team. We had all used TypeScript before, so it would have let us move faster at the start. Rust offered an opportunity to explore its optimization options, and we spent more time figuring out how to set up its server.

For us, Go is the middle ground. It gives us a new technology to learn, and we got the first route working in 12 minutes with the standard library. We accept that getting up to speed will take longer than with TypeScript.
