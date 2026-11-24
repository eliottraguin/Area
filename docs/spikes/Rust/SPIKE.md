# Rust Spike for the AREA Bootstrap

**Timebox** — 25 minutes, the same as for the other two spikes. A minimal HTTP route is enough to compare the initial learning curve, server startup, and port configuration.

## Result

I reached the Rust spike target in **20 minutes**.

- I set up an HTTP server with Axum for routing and Tokio for asynchronous execution.
- The server reads `PORT` with `std::env::var` and defaults to `8080`.
- `GET /ping` returns exactly `{"status":"ok"}`, with no trailing newline. I also added a `GET /hello` route.

## Challenges

Figuring out how to set up the server in Rust took the most time.

## Next Steps

I would add an automated HTTP test to check the status and exact response body of `/ping`, then test a call to an external API from an asynchronous handler.

## Reproduce the Spike

From `docs/spikes/Rust`, run `PORT=8080 cargo run --locked`, then run `curl http://localhost:8080/ping` in a second terminal.

The prototype was built with `cargo build --locked --offline`. An HTTP check on another available port confirmed that the server uses `PORT` and that the `/ping` response body matches the expected result exactly.
