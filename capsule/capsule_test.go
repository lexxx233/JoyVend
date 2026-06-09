package capsule_test

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mykeep.ai/capsule"
)

// req builds a loopback request (the server guard enforces loopback socket + Host).
func req(method, path, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.RemoteAddr = "127.0.0.1:54321"
	r.Host = "127.0.0.1:8765"
	return r
}

func newDEK(t *testing.T) []byte {
	t.Helper()
	dek := make([]byte, 32)
	if _, err := rand.Read(dek); err != nil {
		t.Fatal(err)
	}
	return dek
}

// dup copies a DEK so the component can adopt+wipe its own slice without disturbing
// the test's copy (mirrors the aggregator handing each component a fresh sub-key).
func dup(b []byte) []byte { return append([]byte(nil), b...) }

// TestComponentInjectedDEKRoundTrip proves capsule can be unlocked with an externally
// supplied key (no password), serves its API on a shared mux, and that data sealed
// under that key survives a Lock + reopen with the same key.
func TestComponentInjectedDEKRoundTrip(t *testing.T) {
	t.Setenv("MYKEEP_EMBEDDER", "hash")
	dir := t.TempDir()
	dek := newDEK(t)

	c, err := capsule.New(capsule.Options{DataDir: dir, Portable: true, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !c.FirstLaunch() {
		t.Fatal("expected first launch on a fresh dir")
	}
	if err := c.Unlock(context.Background(), dup(dek)); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	mux := http.NewServeMux()
	c.Mount(mux)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req("GET", "/v1/health", ""))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"status"`) {
		t.Fatalf("health => %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req("POST", "/v1/banks/default/retain",
		`{"items":[{"content":"Alice prefers oat milk","type":"experience"}]}`))
	if w.Code != 200 {
		t.Fatalf("retain => %d %s", w.Code, w.Body.String())
	}

	if err := c.Lock(); err != nil {
		t.Fatalf("lock: %v", err)
	}

	// Reopen the SAME data dir with the SAME key — the memory must survive.
	c2, err := capsule.New(capsule.Options{DataDir: dir, Portable: true, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if c2.FirstLaunch() {
		t.Fatal("second open should not be first launch")
	}
	if err := c2.Unlock(context.Background(), dup(dek)); err != nil {
		t.Fatalf("reopen unlock: %v", err)
	}
	defer c2.Lock()
	mux2 := http.NewServeMux()
	c2.Mount(mux2)

	w = httptest.NewRecorder()
	mux2.ServeHTTP(w, req("GET", "/v1/banks/default/memories?limit=10", ""))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Alice prefers oat milk") {
		t.Fatalf("memories after reopen => %d %s", w.Code, w.Body.String())
	}
}

// TestComponentWrongDEKFails proves the store is genuinely keyed by the injected DEK:
// a different key cannot open a store sealed under the original.
func TestComponentWrongDEKFails(t *testing.T) {
	t.Setenv("MYKEEP_EMBEDDER", "hash")
	dir := t.TempDir()
	dek := newDEK(t)

	c, _ := capsule.New(capsule.Options{DataDir: dir, Version: "test"})
	if err := c.Unlock(context.Background(), dup(dek)); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	mux := http.NewServeMux()
	c.Mount(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req("POST", "/v1/banks/default/retain",
		`{"items":[{"content":"sealed under the original key","type":"experience"}]}`))
	if w.Code != 200 {
		t.Fatalf("retain => %d %s", w.Code, w.Body.String())
	}
	if err := c.Lock(); err != nil {
		t.Fatalf("lock: %v", err)
	}

	wrong := newDEK(t)
	c2, _ := capsule.New(capsule.Options{DataDir: dir, Version: "test"})
	if err := c2.Unlock(context.Background(), wrong); err == nil {
		_ = c2.Lock()
		t.Fatal("expected unlock with a wrong DEK to fail")
	}
}
