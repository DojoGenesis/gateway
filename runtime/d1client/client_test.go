package d1client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew_DefaultBaseURL(t *testing.T) {
	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "tok-1",
	})
	if c.baseURL != defaultBaseURL {
		t.Errorf("expected default base URL %q, got %q", defaultBaseURL, c.baseURL)
	}
}

func TestNew_CustomBaseURL(t *testing.T) {
	custom := "https://custom.example.com/v1"
	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "tok-1",
		BaseURL:    custom,
	})
	if c.baseURL != custom {
		t.Errorf("expected custom base URL %q, got %q", custom, c.baseURL)
	}
}

func TestQuery_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %s", ct)
		}

		// Verify request body
		var req queryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.SQL != "SELECT * FROM users WHERE id = ?" {
			t.Errorf("unexpected SQL: %s", req.SQL)
		}

		// Return a valid D1 response
		resp := QueryResponse{
			Success: true,
			Result: []QueryResult{
				{
					Results: []map[string]any{
						{"id": float64(1), "name": "alice"},
						{"id": float64(2), "name": "bob"},
					},
					Success: true,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "test-token",
		BaseURL:    ts.URL,
	})

	rows, err := c.Query(context.Background(), "SELECT * FROM users WHERE id = ?", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["name"] != "alice" {
		t.Errorf("expected alice, got %v", rows[0]["name"])
	}
}

func TestQuery_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Success: false,
			Errors: []APIError{
				{Code: 7500, Message: "syntax error in SQL"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "tok",
		BaseURL:    ts.URL,
	})

	_, err := c.Query(context.Background(), "BAD SQL")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if got := err.Error(); got != "d1: syntax error in SQL (code 7500)" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestQuery_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer ts.Close()

	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "tok",
		BaseURL:    ts.URL,
	})

	_, err := c.Query(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestQuery_NoErrorDetail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Success: false,
			Errors:  []APIError{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "tok",
		BaseURL:    ts.URL,
	})

	_, err := c.Query(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error for failed query with no detail")
	}
	if got := err.Error(); got != "d1: query failed (no error detail)" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestQuery_EmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Success: true,
			Result:  []QueryResult{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "tok",
		BaseURL:    ts.URL,
	})

	rows, err := c.Query(context.Background(), "SELECT * FROM empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows != nil {
		t.Errorf("expected nil rows for empty result, got %v", rows)
	}
}

func TestExec_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Success: true,
			Result: []QueryResult{
				{Results: nil, Success: true},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "tok",
		BaseURL:    ts.URL,
	})

	_, err := c.Exec(context.Background(), "INSERT INTO users (name) VALUES (?)", "charlie")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExec_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QueryResponse{
			Success: false,
			Errors:  []APIError{{Code: 1000, Message: "table not found"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := New(Config{
		AccountID:  "acct-1",
		DatabaseID: "db-1",
		APIToken:   "tok",
		BaseURL:    ts.URL,
	})

	_, err := c.Exec(context.Background(), "DELETE FROM nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Helper function tests ---

func TestString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"non-string int", 42, ""},
		{"non-string float", 3.14, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := String(tt.in)
			if got != tt.want {
				t.Errorf("String(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestInt64(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int64
	}{
		{"nil", nil, 0},
		{"float64", float64(42), 42},
		{"int64", int64(99), 99},
		{"int", int(7), 7},
		{"string", "not-a-number", 0},
		{"negative float64", float64(-5), -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Int64(tt.in)
			if got != tt.want {
				t.Errorf("Int64(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractChanges_EmptyResult(t *testing.T) {
	qr := QueryResponse{Result: []QueryResult{}}
	got := extractChanges(qr)
	if got != 0 {
		t.Errorf("expected 0 changes, got %d", got)
	}
}

func TestURLConstruction(t *testing.T) {
	// Verify the URL format by capturing what the client builds
	var capturedURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.Path
		resp := QueryResponse{Success: true, Result: []QueryResult{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := New(Config{
		AccountID:  "my-acct",
		DatabaseID: "my-db",
		APIToken:   "tok",
		BaseURL:    ts.URL,
	})

	c.Query(context.Background(), "SELECT 1")
	expected := "/accounts/my-acct/d1/database/my-db/query"
	if capturedURL != expected {
		t.Errorf("expected URL path %q, got %q", expected, capturedURL)
	}
}
