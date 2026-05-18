package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	aaa "yadro.com/course/api/adapters/aaa"
	"yadro.com/course/api/core"
)

func encodeReply(w io.Writer, reply any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(reply); err != nil {
		return fmt.Errorf("could not encode reply: %v", err)
	}
	return nil
}

type loginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func NewLoginHandler(log *slog.Logger, auth aaa.AAA) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Error("failed to close request body", "error", err)
			}
		}()

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode login request", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		token, err := auth.Login(req.Name, req.Password)
		if err != nil {
			log.Warn("login failed", "user", req.Name, "error", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := io.WriteString(w, token); err != nil {
			log.Error("failed to write login response", "error", err)
		}
	}
}

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		reply := PingResponse{
			Replies: make(map[string]string),
		}

		for name, pinger := range pingers {
			if err := pinger.Ping(r.Context()); err != nil {
				log.Error("service is not available", "service", name, "error", err)
				reply.Replies[name] = "unavailable"
				continue
			}
			reply.Replies[name] = "ok"
		}

		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode ping reply", "error", err)
		}
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Update(r.Context()); err != nil {
			log.Error("error while update", "error", err)

			if errors.Is(err, core.ErrAlreadyExists) {
				http.Error(w, err.Error(), http.StatusAccepted)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type UpdateStats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		stats, err := updater.Stats(r.Context())
		if err != nil {
			log.Error("error while stats", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reply := UpdateStats{
			WordsTotal:    stats.WordsTotal,
			WordsUnique:   stats.WordsUnique,
			ComicsFetched: stats.ComicsFetched,
			ComicsTotal:   stats.ComicsTotal,
		}

		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode stats reply", "error", err)
		}
	}
}

type UpdateStatus struct {
	Status string `json:"status"`
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status, err := updater.Status(r.Context())
		if err != nil {
			log.Error("error while status", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reply := UpdateStatus{
			Status: string(status),
		}

		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode status reply", "error", err)
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Drop(r.Context()); err != nil {
			log.Error("error while drop", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type searchResponseComics struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type searchResponse struct {
	Comics []searchResponseComics `json:"comics"`
	Total  int                    `json:"total"`
}

const defaultSearchLimit = 10

type searchFunc func(ctx context.Context, phrase string, limit int) ([]core.Comics, error)

func newSearchHTTPHandler(log *slog.Logger, doSearch searchFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()

		phrase := strings.TrimSpace(q.Get("phrase"))
		if phrase == "" {
			http.Error(w, "phrase is required", http.StatusBadRequest)
			return
		}

		limitStr := strings.TrimSpace(q.Get("limit"))
		limit := defaultSearchLimit
		if limitStr != "" {
			n, err := strconv.Atoi(limitStr)
			if err != nil || n <= 0 {
				http.Error(w, "limit must be positive integer", http.StatusBadRequest)
				return
			}
			limit = n
		}

		comics, err := doSearch(ctx, phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				writeSearchResponse(w, log, searchResponse{
					Comics: []searchResponseComics{},
					Total:  0,
				})
				return
			}

			log.Error("search failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := searchResponse{
			Comics: make([]searchResponseComics, 0, len(comics)),
			Total:  len(comics),
		}
		for _, c := range comics {
			resp.Comics = append(resp.Comics, searchResponseComics{
				ID:  c.ID,
				URL: c.URL,
			})
		}

		writeSearchResponse(w, log, resp)
	}
}

func writeSearchResponse(w http.ResponseWriter, log *slog.Logger, resp searchResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("failed to encode search response", "error", err)
	}
}

func NewSearchHandler(log *slog.Logger, searchClient core.Searcher) http.Handler {
	return newSearchHTTPHandler(log, func(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
		return searchClient.Search(ctx, phrase, limit)
	})
}

func NewIndexSearchHandler(log *slog.Logger, searchClient core.Searcher) http.Handler {
	return newSearchHTTPHandler(log, func(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
		return searchClient.Search(ctx, phrase, limit)
	})
}
