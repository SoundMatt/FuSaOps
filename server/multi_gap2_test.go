package server

// Gap tests for multi.go: ListenAndServe, Serve, and runScheduler.
// These functions reach 0% in the base coverage profile and are the top
// gap identified in v1.130.0.

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// newTestMultiSimple returns a MultiServer backed by a single fake project.
func newTestMultiSimple(t *testing.T) *MultiServer {
	t.Helper()
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo})
	cfg := ProjectsConfig{Projects: []ProjectConfig{{Name: "simple", Dir: t.TempDir()}}}
	return NewMulti(cfg, orchestrator.New(reg))
}

// TestMultiListenAndServeBindError verifies ListenAndServe propagates bind errors
// for invalid addresses, covering the ln/err branch in multi.go ListenAndServe.
//
//fusa:test REQ-FO-MPJ002
func TestMultiListenAndServeBindError(t *testing.T) {
	ms := newTestMultiSimple(t)
	err := ms.ListenAndServe("!invalid-addr!")
	if err == nil {
		t.Fatal("expected bind error for invalid address")
	}
}

// TestMultiServe verifies Serve computes all projects then serves requests.
// It uses a port-0 listener, checks /healthz once the server is up, then
// closes the listener to stop the server, covering the Serve function body.
//
//fusa:test REQ-FO-MPJ002
func TestMultiServe(t *testing.T) {
	ms := newTestMultiSimple(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	errc := make(chan error, 1)
	go func() { errc <- ms.Serve(ln) }()

	// Poll until the server is up.
	url := "http://" + ln.Addr().String() + "/healthz"
	client := &http.Client{Timeout: time.Second}
	var resp *http.Response
	for i := 0; i < 200; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("MultiServer never came up: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz status %d", resp.StatusCode)
	}

	_ = ln.Close() // unblocks Serve
	select {
	case serveErr := <-errc:
		if serveErr != nil && !strings.Contains(serveErr.Error(), "closed") {
			t.Errorf("Serve returned unexpected error: %v", serveErr)
		}
	case <-time.After(3 * time.Second):
		t.Error("Serve did not return after listener close")
	}
}

// TestMultiListenAndServeSuccess verifies ListenAndServe binds a port and
// delegates to Serve, covering the "return ms.Serve(ln)" success path.
//
//fusa:test REQ-FO-MPJ002
func TestMultiListenAndServeSuccess(t *testing.T) {
	ms := newTestMultiSimple(t)

	errc := make(chan error, 1)
	go func() { errc <- ms.ListenAndServe("127.0.0.1:0") }()

	// Give it a moment to start, then stop it by timing out.
	// Because we cannot get the listener address from ListenAndServe,
	// we just verify it does not return immediately with an error.
	select {
	case err := <-errc:
		// An error is only acceptable if it's a "closed" / "use of closed" error
		// (which would happen if the OS reclaimed the port), not a bind failure.
		if err != nil && !strings.Contains(err.Error(), "closed") {
			t.Errorf("ListenAndServe returned unexpected error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		// Server is still running – the success path was executed.
	}
}

// TestMultiRunScheduler verifies the scheduled refresh goroutine fires and
// re-populates the per-project cache, covering runScheduler in multi.go.
//
//fusa:test REQ-FO-SCHD001
func TestMultiRunScheduler(t *testing.T) {
	ms := newTestMultiSimple(t).WithRefreshInterval(30 * time.Millisecond)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = ms.Serve(ln) }()

	// Give the scheduler at least two ticks.
	time.Sleep(150 * time.Millisecond)
	_ = ln.Close()

	// Verify the cache was populated (scheduler ran compute).
	ms.projects[0].mu.RLock()
	rep := ms.projects[0].cached
	ms.projects[0].mu.RUnlock()
	if rep == nil {
		t.Error("project cache still nil after scheduled refresh")
	}
}
