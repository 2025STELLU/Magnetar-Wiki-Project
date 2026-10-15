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

// ForumPostsPrefix is the JSON API root for the console forum-post manager.
const ForumPostsPrefix = "/pw-api/console/forum-posts/"

// ConsoleForumPosts serves the forum-post screen of the new console.
// Wiring (mux registration) is done by the parent; this file only defines
// the handler.
type ConsoleForumPosts struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleForumPosts)(nil)

func NewConsoleForumPosts(d Deps, next http.Handler) *ConsoleForumPosts {
	return &ConsoleForumPosts{deps: d, next: next}
}

func (h *ConsoleForumPosts) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, ForumPostsPrefix)
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
	case head == "categories" && r.Method == http.MethodGet:
		h.categories(w, r, loc)
	case head == "delete" && r.Method == http.MethodPost:
		h.delete(w, r, loc)
	case head == "lock" && r.Method == http.MethodPost:
		h.lock(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// forumPostsAllowed mirrors Console.allowed: login + ManageForum, matching
// the old admin forum-posts screen.
func (h *ConsoleForumPosts) forumPostsAllowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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
	if !perms.Resolve(subject, nil).Has(perms.ManageForum) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

func forumPostListJSON(rows []db.AdminForumPostRow, total int) wikijson.Object {
	items := make(wikijson.Array, 0, len(rows))
	for _, p := range rows {
		items = append(items, wikijson.Object{
			{Key: "id", Value: p.ID},
			{Key: "title", Value: p.Thread},
			{Key: "author", Value: p.Author},
			{Key: "category", Value: p.Category},
			{Key: "locked", Value: p.Locked},
			{Key: "created_at", Value: p.CreatedAt.Format(time.RFC3339)},
		})
	}
	return wikijson.Object{
		{Key: "posts", Value: items},
		{Key: "total", Value: total},
	}
}

func (h *ConsoleForumPosts) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.forumPostsAllowed(w, r, loc) {
		return
	}
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	var categoryID int64
	if n, err := strconv.ParseInt(q.Get("category_id"), 10, 64); err == nil && n > 0 {
		categoryID = n
	}
	limit := 20
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}
	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n >= 0 {
		offset = n
	}
	rows, total, err := h.deps.DB.AdminForumPosts(r.Context(), siteID(r.Context()), query, categoryID, limit, offset)
	if err != nil {
		h.deps.log().Error("list forum posts", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(forumPostListJSON(rows, total))
	if err != nil {
		h.deps.log().Error("marshal forum post list", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *ConsoleForumPosts) categories(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.forumPostsAllowed(w, r, loc) {
		return
	}
	cats, err := h.deps.DB.ForumCategories(r.Context(), siteID(r.Context()))
	if err != nil {
		h.deps.log().Error("list forum categories", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	items := make(wikijson.Array, 0, len(cats))
	for _, c := range cats {
		items = append(items, wikijson.Object{
			{Key: "id", Value: c.ID},
			{Key: "name", Value: c.Name},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "categories", Value: items}})
	if err != nil {
		h.deps.log().Error("marshal forum categories", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *ConsoleForumPosts) detail(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.forumPostsAllowed(w, r, loc) {
		return
	}
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	p, err := h.deps.DB.AdminForumPostDetail(r.Context(), siteID(r.Context()), id)
	if err != nil {
		h.deps.log().Error("read forum post", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	detail := wikijson.Object{
		{Key: "id", Value: p.ID},
		{Key: "title", Value: p.Thread},
		{Key: "author", Value: p.Author},
		{Key: "category", Value: p.Category},
		{Key: "summary", Value: p.Summary},
		{Key: "post_count", Value: p.PostCount},
		{Key: "locked", Value: p.Locked},
		{Key: "created_at", Value: p.CreatedAt.Format(time.RFC3339)},
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "post", Value: detail}})
	if err != nil {
		h.deps.log().Error("marshal forum post detail", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type forumPostIDBody struct {
	ID int64 `json:"id"`
}

func (h *ConsoleForumPosts) delete(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	if !h.forumPostsAllowed(w, r, loc) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body forumPostIDBody
	if err := json.Unmarshal(raw, &body); err != nil || body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	// Resolve first so a post from another site 404s instead of deleting.
	if _, err := h.deps.DB.ForumPostThreadID(ctx, siteID(ctx), body.ID); err != nil {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-bad-request")))
		return
	}
	if err := h.deps.DB.DeleteForumPost(ctx, body.ID); err != nil {
		h.deps.log().Error("delete forum post", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeOK(w, loc)
}

type forumPostLockBody struct {
	ID     int64 `json:"id"`
	Locked bool  `json:"locked"`
}

func (h *ConsoleForumPosts) lock(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	if !h.forumPostsAllowed(w, r, loc) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body forumPostLockBody
	if err := json.Unmarshal(raw, &body); err != nil || body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	threadID, err := h.deps.DB.ForumPostThreadID(ctx, siteID(ctx), body.ID)
	if errors.Is(err, db.ErrNotFound) || err != nil {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-bad-request")))
		return
	}
	if err := h.deps.DB.SetForumThreadLocked(ctx, siteID(ctx), threadID, body.Locked); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, field("error", loc.T("api-bad-request")))
			return
		}
		h.deps.log().Error("lock forum thread", "id", threadID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeOK(w, loc)
}

// writeOK emits {"ok": true} with a real boolean.
func writeOK(w http.ResponseWriter, loc *i18n.Localizer) {
	resp, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
