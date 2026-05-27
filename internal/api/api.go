package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/nathanmauro/agent-observatory/internal/db"
	"github.com/nathanmauro/agent-observatory/internal/indexer"
	"github.com/nathanmauro/agent-observatory/internal/models"
	"github.com/nathanmauro/agent-observatory/internal/processes"
	"github.com/nathanmauro/agent-observatory/internal/ws"
)

const version = "0.1.0"

func NewRouter(database *db.DB, ix *indexer.Indexer, broker *ws.Broker, mon *processes.Monitor) chi.Router {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/agents", handleListAgents(database))
		r.Get("/sessions", handleListSessions(database))
		r.Get("/sessions/{id}", handleGetSession(database))
		r.Get("/search", handleSearch(database))
		r.Get("/memory", handleListMemory(database))
		r.Get("/memory/{id}", handleGetMemory(database))
		r.Get("/timeline", handleListTimeline(database))
		r.Get("/stats", handleGetStats(database))
		r.Get("/processes", handleListProcesses(mon))
		r.Post("/reindex", handleReindex(ix))
		r.Get("/ws", broker.HandleWS)
	})

	r.Handle("/*", staticHandler())

	return r
}

func handleListAgents(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agents, err := database.ListAgents(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if agents == nil {
			agents = []models.Agent{}
		}
		writeJSON(w, http.StatusOK, agents)
	}
}

func handleListSessions(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := queryInt(r, "limit", 50)
		if limit > 200 {
			limit = 200
		}

		resp, err := database.ListSessions(r.Context(),
			r.URL.Query().Get("agent"),
			r.URL.Query().Get("project"),
			r.URL.Query().Get("q"),
			limit,
			r.URL.Query().Get("cursor"),
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if resp.Data == nil {
			resp.Data = []models.Session{}
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleGetSession(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		session, err := database.GetSession(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if session == nil {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}

		events, err := database.GetSessionEvents(r.Context(), id, 100, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if events == nil {
			events = &models.PagedResponse[models.SessionEvent]{Data: []models.SessionEvent{}}
		}
		if events.Data == nil {
			events.Data = []models.SessionEvent{}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"session":     session,
			"events":      events.Data,
			"next_cursor": events.NextCursor,
		})
	}
}

func handleSearch(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			writeError(w, http.StatusBadRequest, "q parameter is required")
			return
		}
		limit := queryInt(r, "limit", 50)
		if limit > 200 {
			limit = 200
		}

		results, err := database.Search(r.Context(), q, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []models.SearchResult{}
		}
		writeJSON(w, http.StatusOK, results)
	}
}

func handleListMemory(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := queryInt(r, "limit", 50)
		if limit > 200 {
			limit = 200
		}
		resp, err := database.ListMemoryDocs(r.Context(),
			r.URL.Query().Get("agent"),
			r.URL.Query().Get("project"),
			r.URL.Query().Get("q"),
			limit,
			r.URL.Query().Get("cursor"),
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if resp.Data == nil {
			resp.Data = []models.MemoryDoc{}
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleGetMemory(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		doc, err := database.GetMemoryDoc(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if doc == nil {
			writeError(w, http.StatusNotFound, "memory doc not found")
			return
		}
		writeJSON(w, http.StatusOK, doc)
	}
}

func handleListTimeline(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := queryInt(r, "limit", 50)
		if limit > 200 {
			limit = 200
		}
		resp, err := database.ListTimelineItems(r.Context(),
			r.URL.Query().Get("agent"),
			r.URL.Query().Get("kind"),
			limit,
			r.URL.Query().Get("cursor"),
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if resp.Data == nil {
			resp.Data = []models.TimelineItem{}
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleGetStats(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := database.GetStats(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, stats)
	}
}

func handleListProcesses(mon *processes.Monitor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		procs := mon.Snapshot()
		if procs == nil {
			procs = []models.Process{}
		}
		writeJSON(w, http.StatusOK, procs)
	}
}

func handleReindex(ix *indexer.Indexer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			if err := ix.IndexAll(context.Background()); err != nil {
				log.Printf("reindex error: %v", err)
			}
		}()
		writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || isLocalhostOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalhostOrigin(origin string) bool {
	for _, prefix := range []string{"http://localhost", "http://127.0.0.1", "http://[::1]"} {
		if len(origin) >= len(prefix) && origin[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
	})
}
