package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func SendJsonResponse(w http.ResponseWriter, statusCode int, res any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		SendJsonErrorMsg(w, http.StatusInternalServerError, "failed to encode JSON response")
	}
}

func SendJsonError(w http.ResponseWriter, statusCode int, err error) {
	type errMsg struct {
		Error string `json:"error"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errMsg{Error: err.Error()})
}

func SendJsonErrorMsg(w http.ResponseWriter, statusCode int, msg string) {
	SendJsonError(w, statusCode, errors.New(msg))
}

func SendErrorWebsocket(ws *websocket.Conn, msg string) {
	closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, msg)
	_ = ws.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(time.Second))
	_ = ws.Close()
}
