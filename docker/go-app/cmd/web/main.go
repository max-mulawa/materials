package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"api.com/quick/pkg/messages"
	stg "api.com/quick/pkg/storage"
	"api.com/quick/pkg/storage/pg"
)

const (
	HttpPort = 8080
)

func main() {

	var storage stg.Storage
	storage, err := pg.New("postgres://api:pass@localhost:5432/msg")
	if err != nil {
		slog.Error("storage init failed", "err", err)
	}

	healthz := func(w http.ResponseWriter, r *http.Request) {
		slog.Info("healtz endpoint called")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status":"ok"}`)
	}
	http.HandleFunc("GET /healthz", healthz)

	http.HandleFunc("GET /messages", func(w http.ResponseWriter, r *http.Request) {
		msgs, err := storage.All()
		if err != nil {
			slog.Error("get messages failed", "err", err)
			http.Error(w, "get meesages failed", http.StatusInternalServerError)
			return
		}

		payload, err := json.Marshal(msgs)
		if err != nil {
			slog.Error("messages marshaling failed", "err", err)
			http.Error(w, "meesages marshalling failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	})

	http.HandleFunc("POST /messages", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("body read failed", "err", err)
			http.Error(w, "body read failed failed", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		msg := messages.Message{}
		err = json.Unmarshal(body, &msg)
		if err != nil {
			slog.Error("body unmarshal failed", "err", err)
			http.Error(w, "body unmarshal failed", http.StatusBadRequest)
			return
		}

		err = storage.Store(msg)
		if err != nil {
			slog.Error("store failed", "err", err)
			http.Error(w, "store failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("GET /messages/{id}", func(w http.ResponseWriter, r *http.Request) {
		//r.PathValue("id")
	})

	slog.Info("Server listening", "port", HttpPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", HttpPort), nil)
	if err != nil {
		slog.Error("server failed to start", "err", err)
		os.Exit(1)
	}
}
