package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type server struct {
	cfg       Config
	log       *slog.Logger
	api       *APIClient
	templates *template.Template
}

func newServer(cfg Config, log *slog.Logger, api *APIClient, templates *template.Template) *server {
	if log == nil {
		log = slog.Default()
	}
	return &server{
		cfg:       cfg,
		log:       log,
		api:       api,
		templates: templates,
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleSearch)
	mux.HandleFunc("GET /admin", s.handleAdmin)
	mux.HandleFunc("POST /admin/login", s.handleLogin)
	mux.HandleFunc("POST /admin/logout", s.handleLogout)
	mux.HandleFunc("POST /admin/update", s.handleUpdate)
	mux.HandleFunc("POST /admin/drop", s.handleDrop)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

type searchPageData struct {
	Phrase string
	Limit  int
	Mode   string

	Comics []Comics
	Total  int

	Message string
	Error   string
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	const defaultLimit = 10

	q := r.URL.Query()
	phrase := strings.TrimSpace(q.Get("phrase"))
	mode := q.Get("mode")
	if mode != "index" {
		mode = "live"
	}

	limit := defaultLimit
	if limitStr := strings.TrimSpace(q.Get("limit")); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	data := searchPageData{
		Phrase: phrase,
		Limit:  limit,
		Mode:   mode,
	}

	if phrase != "" {
		ctx, cancel := s.withTimeout(r.Context())
		defer cancel()

		reply, err := s.api.Search(ctx, phrase, limit, mode == "index")
		if err != nil {
			data.Error = err.Error()
		} else {
			data.Comics = reply.Comics
			data.Total = reply.Total
			if reply.Total == 0 {
				data.Message = "Ничего не нашли — попробуйте другую формулировку"
			}
		}
	}

	s.render(w, "search.body", data)
}

type adminPageData struct {
	LoggedIn bool
	Status   string
	Stats    *UpdateStats
	TokenTTL time.Duration
	Message  string
	Error    string
}

func (s *server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	msg := strings.TrimSpace(r.URL.Query().Get("msg"))
	errMsg := strings.TrimSpace(r.URL.Query().Get("err"))

	data := adminPageData{
		LoggedIn: s.tokenFromCookie(r) != "",
		TokenTTL: s.cfg.TokenTTL,
		Message:  msg,
		Error:    errMsg,
		Status:   "unknown",
	}

	ctx, cancel := s.withTimeout(r.Context())
	defer cancel()

	if stats, err := s.api.Stats(ctx); err == nil {
		data.Stats = &stats
	} else if data.Error == "" {
		data.Error = err.Error()
	}

	if status, err := s.api.Status(ctx); err == nil {
		data.Status = status
	} else if data.Error == "" {
		data.Error = err.Error()
	}

	s.render(w, "admin.body", data)
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.redirectWithMessage(w, r, "/admin", "", "Не удалось разобрать форму")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	password := strings.TrimSpace(r.FormValue("password"))
	if name == "" || password == "" {
		s.redirectWithMessage(w, r, "/admin", "", "Введите логин и пароль")
		return
	}

	ctx, cancel := s.withTimeout(r.Context())
	defer cancel()

	token, err := s.api.Login(ctx, name, password)
	if err != nil {
		s.redirectWithMessage(w, r, "/admin", "", err.Error())
		return
	}

	s.saveToken(w, token)
	s.redirectWithMessage(w, r, "/admin", "Вы успешно вошли", "")
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.dropToken(w)
	s.redirectWithMessage(w, r, "/admin", "Токен удалён", "")
}

func (s *server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	token := s.tokenFromCookie(r)
	if token == "" {
		s.redirectWithMessage(w, r, "/admin", "", "Авторизуйтесь, чтобы запустить обновление")
		return
	}

	ctx, cancel := s.withTimeout(r.Context())
	defer cancel()

	status, err := s.api.Update(ctx, token)
	if err != nil {
		s.redirectWithMessage(w, r, "/admin", "", err.Error())
		return
	}

	switch status {
	case http.StatusOK:
		s.redirectWithMessage(w, r, "/admin", "Обновление запущено", "")
	case http.StatusAccepted:
		s.redirectWithMessage(w, r, "/admin", "Обновление уже идёт", "")
	case http.StatusUnauthorized:
		s.dropToken(w)
		s.redirectWithMessage(w, r, "/admin", "", "Сессия истекла, войдите заново")
	default:
		s.redirectWithMessage(w, r, "/admin", "", fmt.Sprintf("API вернул статус %d", status))
	}
}

func (s *server) handleDrop(w http.ResponseWriter, r *http.Request) {
	token := s.tokenFromCookie(r)
	if token == "" {
		s.redirectWithMessage(w, r, "/admin", "", "Авторизуйтесь, чтобы очистить базу")
		return
	}

	ctx, cancel := s.withTimeout(r.Context())
	defer cancel()

	status, err := s.api.Drop(ctx, token)
	if err != nil {
		s.redirectWithMessage(w, r, "/admin", "", err.Error())
		return
	}

	switch status {
	case http.StatusOK:
		s.redirectWithMessage(w, r, "/admin", "База очищена, запустите обновление", "")
	case http.StatusUnauthorized:
		s.dropToken(w)
		s.redirectWithMessage(w, r, "/admin", "", "Сессия истекла, войдите заново")
	default:
		s.redirectWithMessage(w, r, "/admin", "", fmt.Sprintf("API вернул статус %d", status))
	}
}

func (s *server) withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	timeout := s.cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return context.WithTimeout(parent, timeout)
}

func (s *server) render(w http.ResponseWriter, bodyTemplate string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var body bytes.Buffer
	if err := s.templates.ExecuteTemplate(&body, bodyTemplate, data); err != nil {
		s.log.Error("failed to render body template", "name", bodyTemplate, "error", err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	payload := struct {
		Body template.HTML
	}{
		Body: template.HTML(body.String()),
	}

	if err := s.templates.ExecuteTemplate(w, "layout", payload); err != nil {
		s.log.Error("failed to render layout", "error", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (s *server) tokenFromCookie(r *http.Request) string {
	c, err := r.Cookie(s.cfg.TokenCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

func (s *server) saveToken(w http.ResponseWriter, token string) {
	ttl := s.cfg.TokenTTL
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.TokenCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
	})
}

func (s *server) dropToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.TokenCookie,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *server) redirectWithMessage(w http.ResponseWriter, r *http.Request, path, msg, errMsg string) {
	q := url.Values{}
	if msg != "" {
		q.Set("msg", msg)
	}
	if errMsg != "" {
		q.Set("err", errMsg)
	}
	target := path
	if encoded := q.Encode(); encoded != "" {
		target = target + "?" + encoded
	}

	if errors.Is(r.Context().Err(), context.Canceled) {
		w.WriteHeader(http.StatusRequestTimeout)
		return
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}
