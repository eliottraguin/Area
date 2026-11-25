package frameworkbenchmark

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gofiber/fiber/v3"
)

type pingBody struct {
	Status string `json:"status"`
}

func writePing(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pingBody{Status: "ok"})
}

func stdlib() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", writePing)
	return mux
}

func chiRouter() http.Handler {
	router := chi.NewRouter()
	router.Get("/ping", writePing)
	return router
}

func chiHumaRouter() http.Handler {
	router := chi.NewRouter()
	config := huma.DefaultConfig("AREA framework benchmark", "1.0.0")
	// Omit the optional schema link so all candidates return the same JSON body.
	config.CreateHooks = nil
	api := humachi.New(router, config)
	type output struct{ Body pingBody }
	huma.Get(api, "/ping", func(_ context.Context, _ *struct{}) (*output, error) {
		return &output{Body: pingBody{Status: "ok"}}, nil
	})
	return router
}

func ginRouter() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, pingBody{Status: "ok"})
	})
	return router
}

type benchmarkServer struct {
	serve    func(net.Listener) error
	shutdown func(context.Context) error
}

func standardServer(handler http.Handler) benchmarkServer {
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 3 * time.Second}
	return benchmarkServer{serve: server.Serve, shutdown: server.Shutdown}
}

func fiberServer() benchmarkServer {
	app := fiber.New(fiber.Config{ReadTimeout: 3 * time.Second})
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.JSON(pingBody{Status: "ok"})
	})
	return benchmarkServer{
		serve: func(listener net.Listener) error {
			return app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true})
		},
		shutdown: app.ShutdownWithContext,
	}
}

var candidates = []struct {
	name  string
	build func() benchmarkServer
}{
	{"net_http", func() benchmarkServer { return standardServer(stdlib()) }},
	{"chi", func() benchmarkServer { return standardServer(chiRouter()) }},
	{"chi_huma", func() benchmarkServer { return standardServer(chiHumaRouter()) }},
	{"gin", func() benchmarkServer { return standardServer(ginRouter()) }},
	{"fiber", fiberServer},
}

func startServer(t testing.TB, server benchmarkServer) (*http.Client, string) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := server.shutdown(ctx); err != nil {
			t.Errorf("could not shut down server: %v", err)
		}
		_ = listener.Close()
		select {
		case err := <-serverDone:
			if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
				t.Errorf("server failed: %v", err)
			}
		case <-ctx.Done():
			t.Error("server did not stop before the deadline")
		}
	})
	transport := &http.Transport{MaxIdleConnsPerHost: 1}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	return client, "http://" + listener.Addr().String() + "/ping"
}

func checkPing(t testing.TB, client *http.Client, url string) {
	t.Helper()
	response, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != `{"status":"ok"}` {
		t.Fatalf("got status %d and body %q", response.StatusCode, body)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("got content type %q", contentType)
	}
}

func TestPingResponse(t *testing.T) {
	for _, candidate := range candidates {
		t.Run(candidate.name, func(t *testing.T) {
			client, url := startServer(t, candidate.build())
			checkPing(t, client, url)
		})
	}
}

func TestChiHumaOpenAPI(t *testing.T) {
	router := chiHumaRouter()
	specResponse := httptest.NewRecorder()
	router.ServeHTTP(specResponse, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	var spec struct {
		Paths map[string]map[string]struct {
			Responses map[string]struct {
				Content map[string]json.RawMessage
			}
		}
	}
	if err := json.Unmarshal(specResponse.Body.Bytes(), &spec); err != nil {
		t.Fatalf("could not parse OpenAPI document: %v", err)
	}
	jsonResponse := spec.Paths["/ping"]["get"].Responses["200"].Content["application/json"]
	if specResponse.Code != http.StatusOK || len(jsonResponse) == 0 {
		t.Fatal("OpenAPI must describe a 200 JSON response for GET /ping")
	}

	docsResponse := httptest.NewRecorder()
	router.ServeHTTP(docsResponse, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if docsResponse.Code != http.StatusOK || !strings.HasPrefix(docsResponse.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected browsable API documentation, got HTTP %d", docsResponse.Code)
	}
}

func BenchmarkPing(b *testing.B) {
	for _, candidate := range candidates {
		b.Run(candidate.name, func(b *testing.B) {
			client, url := startServer(b, candidate.build())
			checkPing(b, client, url)
			for b.Loop() {
				response, err := client.Get(url)
				if err != nil {
					b.Fatal(err)
				}
				_, err = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
				if err != nil || response.StatusCode != http.StatusOK {
					b.Fatalf("GET /ping returned HTTP %d, read error: %v", response.StatusCode, err)
				}
			}
		})
	}
}
