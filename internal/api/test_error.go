package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMaxBytesError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10) // Very small limit

		var data map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			fmt.Printf("Error type: %T\n", err)
			fmt.Printf("Error message: %s\n", err.Error())
		}
	})

	// Send a large body
	largeBody := bytes.Repeat([]byte("x"), 100)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(largeBody))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}
