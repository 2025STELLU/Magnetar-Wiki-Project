package webapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/auth"
	"github.com/2025STELLU/MagnetarWiki/internal/csrf"
	"github.com/2025STELLU/MagnetarWiki/internal/db"
	"github.com/2025STELLU/MagnetarWiki/internal/i18n"
	"github.com/2025STELLU/MagnetarWiki/internal/perms"
	"github.com/2025STELLU/MagnetarWiki/internal/repo"
	"github.com/2025STELLU/MagnetarWiki/internal/site"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

// PagesPrefix is the JSON API root for the console page manager.
const PagesPrefix = "/pw-api/console/pages/"

// ConsolePages serves the page-management screen of the new console.
// Wiring (mux registration) is done by the parent; this file only defines
// the handler.
type ConsolePages struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsolePages)(nil)

func NewConsolePages(d Deps, next http.Handler) *ConsolePages {
	return &ConsolePages{deps: d, next: next}
}

func (h *ConsolePages) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, PagesPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case head == "list" && r.Method == http.MethodGet:
		h.list(w, r, loc)
	case head == "detail" && r.Method == http.MethodGet:
		h.detail(w, r, loc)
	case head == "delete" && r.Method == http.MethodPost:
		h.delete(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// pagesAllowed mirrors Console.allowed: login + the named permission.
// List and detail follow the old admin pages screen (EditArticles);
// delete follows the article delete API (DeleteArticles).
func (h *ConsolePages) pagesAllowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, need string) bool {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return false
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(user, time.Now())
	if err != nil {
		h.deps.log().Error("resolve permissions", "user", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return false
	}
	if !perms.Resolve(subject, nil).Has(need) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

func pageListJSON(rows []db.AdminPageRow, total int) wikijson.Object {
	items := make(wikijson.Array, 0, len(rows))
	for _, p := range rows {
		items = append(items, wikijson.Object{
			{Key: "id", Value: p.ID},
			{Key: "name", Value: p.FullName()},
			{Key: "title", Value: p.Title},
			{Key: "revisions", Value: p.Revisions},
			{Key: "updated_at", Value: p.UpdatedAt.Format(time.RFC3339)},
		})
	}
	return wikijson.Object{
		{Key: "pages", Value: items},
		{Key: "total", Value: total},
	}
}

func (h *ConsolePages) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.pagesAllowed(w, r, loc, perms.EditArticles) {
		return
	}
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	limit := 20
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}
	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n >= 0 {
		offset = n
	}
	rows, total, err := h.deps.DB.AdminPages(r.Context(), siteID(r.Context()), query, "", limit, offset)
	if err != nil {
		h.deps.log().Error("list pages", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(pageListJSON(rows, total))
	if err != nil {
		h.deps.log().Error("marshal page list", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// pageSummary trims the wikitext source to a short plain preview.
func pageSummary(source string) string {
	const maxRunes = 200
	runes := []rune(strings.TrimSpace(source))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "…"
}

func (h *ConsolePages) detail(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.pagesAllowed(w, r, loc, perms.EditArticles) {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	ctx := r.Context()
	article, err := h.deps.DB.ArticleByName(ctx, siteID(ctx), name)
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-page-not-found")))
		return
	}
	if err != nil {
		h.deps.log().Error("read page", "name", name, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	source, err := h.deps.DB.LatestSource(ctx, article.ID)
	if err != nil {
		h.deps.log().Error("read page source", "id", article.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	revisions, err := h.deps.DB.ArticleLogCount(ctx, article.ID)
	if err != nil {
		h.deps.log().Error("count revisions", "id", article.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	detail := wikijson.Object{
		{Key: "id", Value: article.ID},
		{Key: "name", Value: article.FullName()},
		{Key: "title", Value: article.Title},
		{Key: "summary", Value: pageSummary(source)},
		{Key: "revisions", Value: revisions},
		{Key: "locked", Value: article.Locked},
		{Key: "updated_at", Value: article.UpdatedAt.Format(time.RFC3339)},
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "page", Value: detail}})
	if err != nil {
		h.deps.log().Error("marshal page detail", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type deletePageBody struct {
	Name string `json:"name"`
}

func (h *ConsolePages) delete(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-page-not-found")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		refuseCSRF(w, r, loc)
		return
	}
	if !h.pagesAllowed(w, r, loc, perms.DeleteArticles) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body deletePageBody
	if err := json.Unmarshal(raw, &body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	article, err := h.deps.DB.ArticleByName(ctx, siteID(ctx), strings.TrimSpace(body.Name))
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-page-not-found")))
		return
	}
	if err != nil {
		h.deps.log().Error("read page for delete", "name", body.Name, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := h.deps.DB.DeleteArticle(ctx, siteID(ctx), article.ID, article.FullName()); err != nil {
		h.deps.log().Error("delete page", "id", article.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	resp, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal delete result", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
