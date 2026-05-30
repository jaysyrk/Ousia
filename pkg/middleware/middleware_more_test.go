package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRateLimit(t *testing.T) {
	rl := RateLimit(1, 1)
	handler := rl(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	// First request should pass
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Second request should fail (rate limit exceeded)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req)
	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr2.Code)
	}
}

func TestHeaderKeyFunc(t *testing.T) {
	keyFn := HeaderKeyFunc("X-Real-IP")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	if keyFn(req) != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.168.1.1:5432"
	if keyFn(req2) != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1")
	}
}

func TestVerifyWasmHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dummy.wasm")
	os.WriteFile(path, []byte("dummy"), 0644)

	h := sha256.New()
	h.Write([]byte("dummy"))
	expectedHex := hex.EncodeToString(h.Sum(nil))

	err := VerifyWasmHash(path, expectedHex)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = VerifyWasmHash(path, "invalidhex")
	if err == nil {
		t.Errorf("expected error for mismatched hash")
	}

	err = VerifyWasmHash(filepath.Join(dir, "nonexistent"), expectedHex)
	if err == nil {
		t.Errorf("expected error for nonexistent file")
	}
}

func TestWasmMiddleware(t *testing.T) {
	// Attempt to create with nonexistent file
	_, err := NewWasmMiddleware(context.Background(), "nonexistent.wasm", nil)
	if err == nil {
		t.Errorf("expected error loading nonexistent wasm")
	}

	// Create dummy wasm that fails to compile
	dir := t.TempDir()
	path := filepath.Join(dir, "dummy.wasm")
	os.WriteFile(path, []byte("invalid wasm binary"), 0644)
	_, err = NewWasmMiddleware(context.Background(), path, nil)
	if err == nil {
		t.Errorf("expected error compiling invalid wasm")
	}
}

func TestWasmMiddleware_Valid(t *testing.T) {
	wasmPath := "../../plugin.wasm"
	if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
		t.Skip("plugin.wasm not found, skipping valid wasm test")
	}

	var handlerCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	m, err := NewWasmMiddleware(context.Background(), wasmPath, next)
	if err != nil {
		t.Fatalf("failed to load valid wasm: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	m.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Errorf("expected next handler to be called")
	}
}
