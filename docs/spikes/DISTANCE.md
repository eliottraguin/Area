# Distance Covered in the AREA Spikes

We set a **25-minute timebox** for each of the three spikes. This gave us time to attempt a minimal HTTP route and identify what slowed us down as we got started with each stack.

## Comparison Method

We measured progress by counting how many of the brief’s three criteria each prototype met. Each criterion is worth one point.

1. The server can listen on port `8080`.
2. `GET /ping` returns exactly `{"status":"ok"}`, with no trailing newline.
3. The port is configurable through the `PORT` environment variable.

These criteria give all three spikes the same target. The time to first success and the most time-consuming step show the effort behind each score. The score covers only the three criteria above.

## Comparison

| Criterion | Go | Rust | TypeScript |
| --- | --- | --- | --- |
| HTTP stack | `net/http` (standard library) | Axum and Tokio | Express and Node.js |
| Start on port `8080` | `PORT=8080 go run .` | `PORT=8080 cargo run --locked` | `PORT=8080 node main.ts` |
| Exact `GET /ping` response body | Implemented | Implemented | Implemented |
| Port configuration | `os.Getenv("PORT")` | `std::env::var("PORT")`, defaults to `8080` | `process.env.PORT`, defaults to `3000` |
| **Criteria met** | **3/3** | **3/3** | **3/3** |
| **Time to first success** | **12 min** | **20 min** | **5 min** |
| Most time-consuming step | Looking up how to proceed in the Go documentation. | Figuring out how to set up the server. | Refreshing existing TypeScript knowledge. |
| Spike report | [SPIKE.md](Go/SPIKE.md) | [SPIKE.md](Rust/SPIKE.md) | [SPIKE.md](Typescript/SPIKE.md) |

We verified the Rust and TypeScript responses with `curl` on available ports set through `PORT`, because `8080` was in use. We checked the Go handler with `httptest`, which confirmed the `200` status and the exact response body.

## Results

All three prototypes reached the target within 25 minutes. TypeScript was the quickest to set up in these spikes, with a first success in 5 minutes, followed by Go in 12 minutes and Rust in 20 minutes. Prior familiarity with TypeScript made it easier to get started. Consulting the Go documentation took the most time in the Go spike. In the Rust spike, figuring out how to set up the server took the most time.
