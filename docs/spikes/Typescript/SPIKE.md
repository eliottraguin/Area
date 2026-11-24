# TypeScript Spike for the AREA Bootstrap

**Timebox** — 25 minutes, the same as for the other two spikes. A minimal HTTP route is enough to compare the initial learning curve, server startup, and port configuration.

## Result

I reached the TypeScript spike target in **5 minutes**.

- I created an HTTP server with Express and added `Request` and `Response` types to the handlers.
- The server reads `process.env.PORT` and defaults to `3000`, so I start it with `PORT=8080` for the exercise.
- `GET /ping` returns exactly `{"status":"ok"}`, with no trailing newline. `GET /` displays `Hello, World!`.

## Getting Started

I already knew TypeScript, so getting started was quick. Refreshing my knowledge of the language took the most time.

## Next Steps

I would add a start command to `package.json` and an automated HTTP test to check the status and exact response body of `/ping`.

## Reproduce the Spike

From `docs/spikes/Typescript`, install the dependencies with `npm ci`, then run `PORT=8080 node main.ts` with Node.js 24. In a second terminal, run `curl http://localhost:8080/ping`.

An HTTP check on another available port confirmed that the server uses `PORT` and that the `/ping` response body matches the expected result exactly.
