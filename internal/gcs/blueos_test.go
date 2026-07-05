package gcs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerExposesBlueOSRegisterService(t *testing.T) {
	server := NewServer(NewStore("nats://test", true), nil, "", ServerOptions{})
	req := httptest.NewRequest(http.MethodGet, "/register_service", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body["name"] != "SemLink Companion" {
		t.Fatalf("name = %v", body["name"])
	}
	if body["works_in_relative_paths"] != true {
		t.Fatalf("works_in_relative_paths = %v", body["works_in_relative_paths"])
	}
}
