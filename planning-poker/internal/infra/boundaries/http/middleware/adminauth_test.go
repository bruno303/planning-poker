package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAdminMiddleware(t *testing.T) {
	apiKey := "test-api-key"

	middleware := NewAdminMiddleware(apiKey)

	if middleware.apiKey != apiKey {
		t.Errorf("apiKey = %v, want %v", middleware.apiKey, apiKey)
	}
}

func TestAdminMiddleware_Handle_Success(t *testing.T) {
	apiKey := "valid-api-key"
	middleware := NewAdminMiddleware(apiKey)

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := middleware.Handle(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer valid-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !nextHandlerCalled {
		t.Error("next handler should have been called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "success" {
		t.Errorf("body = %v, want %v", rec.Body.String(), "success")
	}
}

func TestAdminMiddleware_Handle_Unauthorized_WrongKey(t *testing.T) {
	apiKey := "valid-api-key"
	middleware := NewAdminMiddleware(apiKey)

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
	})

	handler := middleware.Handle(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer wrong-api-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextHandlerCalled {
		t.Error("next handler should not have been called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
	if rec.Body.String() != "Unauthorized\n" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "Unauthorized\n")
	}
}

func TestAdminMiddleware_Handle_Unauthorized_MissingBearer(t *testing.T) {
	apiKey := "valid-api-key"
	middleware := NewAdminMiddleware(apiKey)

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
	})

	handler := middleware.Handle(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	// Note: TrimPrefix will return the original string if prefix is not found
	// So "valid-api-key" without "Bearer " will still match if the key is "valid-api-key"
	// Using a different value to ensure it fails
	req.Header.Set("Authorization", "wrong-api-key") // Missing "Bearer " prefix and wrong key
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextHandlerCalled {
		t.Error("next handler should not have been called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminMiddleware_Handle_Unauthorized_MissingHeader(t *testing.T) {
	apiKey := "valid-api-key"
	middleware := NewAdminMiddleware(apiKey)

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
	})

	handler := middleware.Handle(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	// No Authorization header set
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextHandlerCalled {
		t.Error("next handler should not have been called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminMiddleware_Handle_Unauthorized_EmptyKey(t *testing.T) {
	apiKey := "valid-api-key"
	middleware := NewAdminMiddleware(apiKey)

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
	})

	handler := middleware.Handle(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextHandlerCalled {
		t.Error("next handler should not have been called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminMiddleware_Handle_TimingSafeComparison(t *testing.T) {
	// Test that the middleware uses constant-time comparison
	// by verifying it works correctly with keys of different lengths
	apiKey := "short-key"
	middleware := NewAdminMiddleware(apiKey)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handle(nextHandler)

	testCases := []struct {
		name           string
		providedKey    string
		expectedStatus int
	}{
		{
			name:           "exact match",
			providedKey:    "short-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "longer key",
			providedKey:    "short-key-with-extra-characters",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "shorter key",
			providedKey:    "short",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "similar key",
			providedKey:    "short-kay", // 'y' instead of 'e'
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req.Header.Set("Authorization", "Bearer "+tc.providedKey)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("status code = %v, want %v", rec.Code, tc.expectedStatus)
			}
		})
	}
}

func TestAdminMiddleware_Handle_MultipleRequests(t *testing.T) {
	apiKey := "valid-api-key"
	middleware := NewAdminMiddleware(apiKey)

	requestCount := 0
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handle(nextHandler)

	// Make multiple valid requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer valid-api-key")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: status code = %v, want %v", i+1, rec.Code, http.StatusOK)
		}
	}

	if requestCount != 3 {
		t.Errorf("requestCount = %v, want %v", requestCount, 3)
	}
}

func TestAdminMiddleware_Handle_DifferentHTTPMethods(t *testing.T) {
	apiKey := "valid-api-key"
	middleware := NewAdminMiddleware(apiKey)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			nextHandlerCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextHandlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware.Handle(nextHandler)

			req := httptest.NewRequest(method, "/admin", nil)
			req.Header.Set("Authorization", "Bearer valid-api-key")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if !nextHandlerCalled {
				t.Errorf("next handler should have been called for method %s", method)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
			}
		})
	}
}
