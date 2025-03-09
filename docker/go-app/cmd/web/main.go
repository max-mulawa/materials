package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"api.com/quick/pkg/messages"
	stg "api.com/quick/pkg/storage"
	"api.com/quick/pkg/storage/pg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	DefaultHttpPort = "8080"

	DefaultUserName = "api"
	DefaultPassword = "pass"
	DefaultHost     = "localhost"
	DefaultDBPort   = "5432"
	DefaultDBName   = "msg"
)

var (
	requestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "go_app_request_total",
		Help: "The total number of processed requests",
	}, []string{"endpoint", "method", "status"})
)

func main() {

	port := Getenv("HTTP_PORT", DefaultHttpPort)

	user := Getenv("DB_USER", DefaultUserName)
	pass := ""
	passFile := os.Getenv("DB_PASSWORD_FILE")
	if passFile != "" {
		content, err := os.ReadFile(passFile)
		if err != nil {
			slog.Error("read pass file failed", "err", err)
			os.Exit(1)
		}
		pass = string(content)
	} else {
		pass = Getenv("DB_PASSWORD", DefaultPassword)

	}
	host := Getenv("DB_HOST", DefaultHost)
	dbPort := Getenv("DB_PORT", DefaultDBPort)

	var storage stg.Storage
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, dbPort, DefaultDBName)
	slog.Info("conn string", "connString", connString)
	storage, err := pg.New(connString)
	if err != nil {
		slog.Error("storage init failed", "err", err)
	}

	healthz := func(w http.ResponseWriter, r *http.Request) {
		slog.Info("healtz endpoint called")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status":"ok"}`)

		requestTotal.WithLabelValues("healthz", "GET", strconv.Itoa(http.StatusOK)).Inc()
	}
	http.HandleFunc("GET /healthz", healthz)

	http.HandleFunc("GET /messages", func(w http.ResponseWriter, r *http.Request) {
		msgs, err := storage.All()
		if err != nil {
			slog.Error("get messages failed", "err", err)
			http.Error(w, "get meesages failed", http.StatusInternalServerError)

			requestTotal.WithLabelValues("messages", "GET", strconv.Itoa(http.StatusInternalServerError)).Inc()
			return
		}

		payload, err := json.Marshal(msgs)
		if err != nil {
			slog.Error("messages marshaling failed", "err", err)
			http.Error(w, "meesages marshalling failed", http.StatusInternalServerError)

			requestTotal.WithLabelValues("messages", "GET", strconv.Itoa(http.StatusInternalServerError)).Inc()
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)

		requestTotal.WithLabelValues("messages", "GET", strconv.Itoa(http.StatusOK)).Inc()
	})

	http.HandleFunc("POST /messages", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("body read failed", "err", err)
			http.Error(w, "body read failed failed", http.StatusInternalServerError)

			requestTotal.WithLabelValues("messages", "POST", strconv.Itoa(http.StatusInternalServerError)).Inc()
			return
		}
		defer r.Body.Close()

		msg := messages.Message{}
		err = json.Unmarshal(body, &msg)
		if err != nil {
			slog.Error("body unmarshal failed", "err", err)
			http.Error(w, "body unmarshal failed", http.StatusBadRequest)

			requestTotal.WithLabelValues("messages", "POST", strconv.Itoa(http.StatusBadRequest)).Inc()
			return
		}

		err = storage.Store(msg)
		if err != nil {
			slog.Error("store failed", "err", err)
			http.Error(w, "store failed", http.StatusInternalServerError)

			requestTotal.WithLabelValues("messages", "POST", strconv.Itoa(http.StatusInternalServerError)).Inc()
			return
		}
		w.WriteHeader(http.StatusCreated)

		requestTotal.WithLabelValues("messages", "POST", strconv.Itoa(http.StatusCreated)).Inc()
	})

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("GET /messages/{id}", func(w http.ResponseWriter, r *http.Request) {
		val := r.PathValue("id")
		if strings.TrimSpace(val) == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			requestTotal.WithLabelValues("messages/{id}", "GET", strconv.Itoa(http.StatusBadRequest)).Inc()
			return
		}

		id, err := strconv.Atoi(val)
		if err != nil {
			http.Error(w, fmt.Sprintf("wrong format id: %s", err), http.StatusBadRequest)
			requestTotal.WithLabelValues("messages/{id}", "GET", strconv.Itoa(http.StatusBadRequest)).Inc()
			return
		}

		msg, err := storage.Load(messages.MsgID(id))
		if err == stg.ErrNotFound {
			http.Error(w, fmt.Sprintf("message with id: %d not found", id), http.StatusNotFound)
			requestTotal.WithLabelValues("messages/{id}", "GET", strconv.Itoa(http.StatusNotFound)).Inc()
			return
		} else if err != nil {
			http.Error(w, fmt.Sprintf("load failed: %s", err), http.StatusInternalServerError)
			requestTotal.WithLabelValues("messages/{id}", "GET", strconv.Itoa(http.StatusInternalServerError)).Inc()
			return
		}

		json, err := json.Marshal(msg)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to marshal: %s", err), http.StatusInternalServerError)
			requestTotal.WithLabelValues("messages/{id}", "GET", strconv.Itoa(http.StatusInternalServerError)).Inc()
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(json)

		requestTotal.WithLabelValues("messages/{id}", "GET", strconv.Itoa(http.StatusOK)).Inc()
	})

	// exact match to /
	http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "<html><body><h1>hello from go-app</h1></body></html>")

		requestTotal.WithLabelValues("/", "GET", strconv.Itoa(http.StatusOK)).Inc()
	})

	slog.Info("Server listening", "port", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		slog.Error("server failed to start", "err", err)
		os.Exit(1)
	}
}

func Getenv(key string, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}
