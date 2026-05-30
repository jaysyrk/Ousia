package observability

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestInitMetrics(t *testing.T) {
	defer func() {
		recover() // ignore double registration panic if called multiple times
	}()
	InitMetrics()
}

func TestAdminAuthMiddleware(t *testing.T) {
	handler := adminAuthMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	req.Header.Set("Authorization", "Bearer wrong")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}

	req.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Healthz should bypass auth
	req = httptest.NewRequest("GET", "/healthz", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetStatsJSON(t *testing.T) {
	ActiveConnections.Set(42)
	stats := GetStatsJSON()
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}
	
	val, ok := stats["ousia_active_connections"]
	if !ok {
		t.Fatal("expected ousia_active_connections in stats")
	}
	
	list := val.([]map[string]interface{})
	if len(list) == 0 {
		t.Fatal("expected items in list")
	}
	
	if list[0]["value"].(float64) != 42 {
		t.Errorf("expected 42, got %v", list[0]["value"])
	}
}

func TestStartAdminServer(t *testing.T) {
	os.Setenv("OUSIA_ADMIN_TOKEN", "test-token")
	defer os.Unsetenv("OUSIA_ADMIN_TOKEN")
	
	StartAdminServer("127.0.0.1:0", func(mux *http.ServeMux) {
		mux.HandleFunc("/custom", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})
}
