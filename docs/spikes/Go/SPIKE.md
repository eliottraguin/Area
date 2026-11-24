# Go Spike for the AREA Bootstrap

**Timebox** — 25 minutes, the same as for Rust and TypeScript. A single HTTP route is enough to compare the initial learning curve and the steps needed to start each server.

## Result

I reached the Go spike target in **12 minutes**.

- I created an HTTP server in Go using `net/http` from the standard library.
- The server reads the port with `os.Getenv("PORT")`. I can start it on the required port with `PORT=8080 go run .`.
- `GET /ping` returns exactly `{"status":"ok"}`, with no trailing newline.
- During the spike, I also added `/hello`, which displays the requested path, and a white homepage titled “Mon projet AREA”.

## Challenges

Looking up how to proceed in the Go documentation took the most time.

I adjusted the `/ping` response format to match the exact JSON. I also identified why zsh displays a `%` when the response ends without a newline.

## Next Steps

I would restrict `/ping` to the `GET` method and add a test that checks the HTTP status code and exact response body. At this stage, `go test ./...` compiles the program, and adding automated tests is still an open task.

## Reproduce the Spike

From `docs/spikes/Go`, run `PORT=8080 go run .`, then run `curl -w '\n' http://localhost:8080/ping` in a second terminal. The `-w` option adds a newline to the terminal output.
