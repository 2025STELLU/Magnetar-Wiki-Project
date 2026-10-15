package webapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/auth"
	"github.com/2025STELLU/MagnetarWiki/internal/db"
	"github.com/2025STELLU/MagnetarWiki/internal/i18n"
	"github.com/2025STELLU/MagnetarWiki/internal/perms"
	"github.com/2025STELLU/MagnetarWiki/internal/repo"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

// AdminLogPrefix is the JSON API root for the console admin log page.
const AdminLogPrefix = "/pw-api/console/admin-log/"

// ConsoleAdminLog serves the admin-log page of the new console.
// Wiring (mux registration) is done by the parent; this file only defines
// the handler.
type ConsoleAdminLog struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleAdminLog)(nil)

func NewConsoleAdminLog(d Deps, next http.Handler) *ConsoleAdminLog {
	return &ConsoleAdminLog{deps: d, next: next}
}

func (h *ConsoleAdminLog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, AdminLogPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case head == "list" && r.Method == http.MethodGet:
		h.list(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// adminLogAllowed mirrors Console.allowed: login + the named permission.
func (h *ConsoleAdminLog) adminLogAllowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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
	if !perms.Resolve(subject, nil).Has(perms.ViewActionsLog) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

func adminLogEntryJSON(row db.AdminNoteRow) wikijson.Object {
	actor := row.User
	if actor == "" {
		actor = row.Stale
	}
	return wikijson.Object{
		{Key: "id", Value: row.ID},
		{Key: "actor", Value: actor},
		{Key: "action", Value: row.Action},
		{Key: "target", Value: row.Target},
		{Key: "detail", Value: row.Label},
		{Key: "created_at", Value: row.CreatedAt.Format(time.RFC3339)},
	}
}

func (h *ConsoleAdminLog) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.adminLogAllowed(w, r, loc) {
		return
	}
	q := r.URL.Query()
	actor := strings.TrimSpace(q.Get("actor"))
	if len(actor) > 64 {
		actor = actor[:64]
	}
	action := strings.TrimSpace(q.Get("action"))
	limit := 50
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}
	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n >= 0 {
		offset = n
	}
	rows, total, err := h.deps.DB.ConsoleAdminLog(r.Context(), siteID(r.Context()), actor, action, limit, offset)
	if err != nil {
		h.deps.log().Error("list admin log", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	items := make(wikijson.Array, 0, len(rows))
	for _, row := range rows {
		items = append(items, adminLogEntryJSON(row))
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "entries", Value: items},
		{Key: "total", Value: total},
	})
	if err != nil {
		h.deps.log().Error("marshal admin log", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}
