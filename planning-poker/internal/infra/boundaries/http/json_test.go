package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendJsonResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   any
		wantBody   string
		wantStatus int
	}{
		{
			name:       "should send successful JSON response with map",
			statusCode: http.StatusOK,
			response:   map[string]string{"message": "success"},
			wantBody:   `{"message":"success"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "should send JSON response with struct",
			statusCode: http.StatusCreated,
			response: struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{ID: 1, Name: "test"},
			wantBody:   `{"id":1,"name":"test"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "should send JSON response with string",
			statusCode: http.StatusAccepted,
			response:   "simple string",
			wantBody:   `"simple string"`,
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "should send JSON response with number",
			statusCode: http.StatusOK,
			response:   42,
			wantBody:   `42`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "should send JSON response with boolean",
			statusCode: http.StatusOK,
			response:   true,
			wantBody:   `true`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "should send JSON response with array",
			statusCode: http.StatusOK,
			response:   []string{"a", "b", "c"},
			wantBody:   `["a","b","c"]`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "should send JSON response with nil",
			statusCode: http.StatusNoContent,
			response:   nil,
			wantBody:   `null`,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			SendJsonResponse(w, tt.statusCode, tt.response)

			if w.Code != tt.wantStatus {
				t.Errorf("SendJsonResponse() status = %v, want %v", w.Code, tt.wantStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("SendJsonResponse() Content-Type = %v, want application/json", contentType)
			}

			var got, want any
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantBody), &want); err != nil {
				t.Fatalf("failed to unmarshal expected body: %v", err)
			}

			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("SendJsonResponse() body = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestSendJsonError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		wantBody   string
		wantStatus int
	}{
		{
			name:       "should send error response with custom error",
			statusCode: http.StatusBadRequest,
			err:        errors.New("invalid input"),
			wantBody:   `{"error":"invalid input"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "should send error response with internal server error",
			statusCode: http.StatusInternalServerError,
			err:        errors.New("database connection failed"),
			wantBody:   `{"error":"database connection failed"}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "should send error response with not found",
			statusCode: http.StatusNotFound,
			err:        errors.New("resource not found"),
			wantBody:   `{"error":"resource not found"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "should send error response with unauthorized",
			statusCode: http.StatusUnauthorized,
			err:        errors.New("unauthorized access"),
			wantBody:   `{"error":"unauthorized access"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "should send error response with empty error message",
			statusCode: http.StatusBadRequest,
			err:        errors.New(""),
			wantBody:   `{"error":""}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			SendJsonError(w, tt.statusCode, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("SendJsonError() status = %v, want %v", w.Code, tt.wantStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("SendJsonError() Content-Type = %v, want application/json", contentType)
			}

			var got, want map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantBody), &want); err != nil {
				t.Fatalf("failed to unmarshal expected body: %v", err)
			}

			if got["error"] != want["error"] {
				t.Errorf("SendJsonError() error message = %v, want %v", got["error"], want["error"])
			}
		})
	}
}

func TestSendJsonErrorMsg(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		msg        string
		wantBody   string
		wantStatus int
	}{
		{
			name:       "should send error message with bad request",
			statusCode: http.StatusBadRequest,
			msg:        "validation failed",
			wantBody:   `{"error":"validation failed"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "should send error message with internal server error",
			statusCode: http.StatusInternalServerError,
			msg:        "unexpected error occurred",
			wantBody:   `{"error":"unexpected error occurred"}`,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "should send error message with conflict",
			statusCode: http.StatusConflict,
			msg:        "resource already exists",
			wantBody:   `{"error":"resource already exists"}`,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "should send error message with forbidden",
			statusCode: http.StatusForbidden,
			msg:        "access denied",
			wantBody:   `{"error":"access denied"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "should send error message with empty string",
			statusCode: http.StatusBadRequest,
			msg:        "",
			wantBody:   `{"error":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "should send error message with special characters",
			statusCode: http.StatusBadRequest,
			msg:        "error with \"quotes\" and \nnewlines",
			wantBody:   `{"error":"error with \"quotes\" and \nnewlines"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			SendJsonErrorMsg(w, tt.statusCode, tt.msg)

			if w.Code != tt.wantStatus {
				t.Errorf("SendJsonErrorMsg() status = %v, want %v", w.Code, tt.wantStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("SendJsonErrorMsg() Content-Type = %v, want application/json", contentType)
			}

			var got, want map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantBody), &want); err != nil {
				t.Fatalf("failed to unmarshal expected body: %v", err)
			}

			if got["error"] != want["error"] {
				t.Errorf("SendJsonErrorMsg() error message = %v, want %v", got["error"], want["error"])
			}
		})
	}
}
