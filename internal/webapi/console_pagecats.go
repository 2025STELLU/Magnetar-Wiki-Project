package webapi

import (
	"encoding/json"
	"io"
	"net/http"
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

// PageCatsPrefix is the JSON API root for the console page-category manager.
const PageCatsPrefix = "/pw-api/console/page-categories/"

type ConsolePageCats struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsolePageCats)(nil)

func NewConsolePageCats(d Deps, next http.Handler) *ConsolePageCats {
	return &ConsolePageCats{deps: d, next: next}
}

func (h *ConsolePageCats) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, PageCatsPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case head == "list" && r.Method == http.MethodGet:
		h.list(w, r, loc)
	case head == "save" && r.Method == http.MethodPost:
		h.save(w, r, loc)
	case head == "delete" && r.Method == http.MethodPost:
		h.remove(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// allowed mirrors ConsoleRoles.allowed: signed in + holds ManageCategories.
func (h *ConsolePageCats) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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
	if !perms.Resolve(subject, nil).Has(perms.ManageCategories) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

func (h *ConsolePageCats) checkCSRF(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
	current := site.FromContext(r.Context())
	if current == nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-page-not-found")))
		return false
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		refuseCSRF(w, r, loc)
		return false
	}
	return true
}

func (h *ConsolePageCats) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc) {
		return
	}
	rows, err := h.deps.DB.ConsolePageCategories(r.Context(), siteID(r.Context()))
	if err != nil {
		h.deps.log().Error("list page categories", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	items := make(wikijson.Array, 0, len(rows))
	for _, row := range rows {
		items = append(items, wikijson.Object{
			{Key: "id", Value: row.ID},
			{Key: "name", Value: row.Name},
			{Key: "description", Value: row.Description},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "categories", Value: items}})
	if err != nil {
		h.deps.log().Error("marshal page categories", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type pageCatSaveBody struct {
	ID          *int64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *ConsolePageCats) save(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.checkCSRF(w, r, loc) {
		return
	}
	if !h.allowed(w, r, loc) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body pageCatSaveBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Description = strings.TrimSpace(body.Description)
	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var id int64
	if body.ID != nil {
		id = *body.ID
	}
	did := db.AdminChanged
	if id == 0 {
		did = db.AdminCreated
	}
	ctx := r.Context()
	if err := h.deps.DB.ConsoleSavePageCategory(ctx, siteID(ctx), id, body.Name, body.Description); err != nil {
		h.deps.log().Error("save page category", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(ctx, h.deps, did, "page-categories", id, body.Name)
	h.writeOK(w, loc)
}

type pageCatDeleteBody struct {
	ID int64 `json:"id"`
}

func (h *ConsolePageCats) remove(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.checkCSRF(w, r, loc) {
		return
	}
	if !h.allowed(w, r, loc) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body pageCatDeleteBody
	if err := json.Unmarshal(raw, &body); err != nil || body.ID == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	ctx := r.Context()
	site := siteID(ctx)
	var label string
	if rows, err := h.deps.DB.ConsolePageCategories(ctx, site); err == nil {
		for _, row := range rows {
			if row.ID == body.ID {
				label = row.Name
				break
			}
		}
	}
	// DeleteCategory drops the overrides and settings rows too.
	if err := h.deps.DB.DeleteCategory(ctx, site, body.ID); err != nil {
		h.deps.log().Error("delete page category", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(ctx, h.deps, db.AdminDeleted, "page-categories", body.ID, label)
	h.writeOK(w, loc)
}

func (h *ConsolePageCats) writeOK(w http.ResponseWriter, loc *i18n.Localizer) {
	okBody, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal page category ok", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, okBody)
}
