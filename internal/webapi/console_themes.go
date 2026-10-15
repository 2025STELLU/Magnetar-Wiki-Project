package webapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/auth"
	"github.com/2025STELLU/MagnetarWiki/internal/csrf"
	"github.com/2025STELLU/MagnetarWiki/internal/i18n"
	"github.com/2025STELLU/MagnetarWiki/internal/perms"
	"github.com/2025STELLU/MagnetarWiki/internal/repo"
	"github.com/2025STELLU/MagnetarWiki/internal/site"
	"github.com/2025STELLU/MagnetarWiki/internal/update"
	"github.com/2025STELLU/MagnetarWiki/internal/version"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

// ConsoleThemesPrefix is the JSON API root for the /-/console themes page.
const ConsoleThemesPrefix = "/pw-api/console/themes/"

type ConsoleThemes struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleThemes)(nil)

func NewConsoleThemes(d Deps, next http.Handler) *ConsoleThemes {
	return &ConsoleThemes{deps: d, next: next}
}

func (h *ConsoleThemes) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, ConsoleThemesPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	switch {
	case rest == "list" && r.Method == http.MethodGet:
		h.getList(w, r, loc)
	case rest == "activate" && r.Method == http.MethodPost:
		h.postActivate(w, r, loc)
	case rest == "check_update" && r.Method == http.MethodPost:
		h.postCheckUpdate(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// allowed mirrors Console.allowed: signed in + holds every named permission.
func (h *ConsoleThemes) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, wanted ...string) bool {
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
	held := perms.Resolve(subject, nil)
	for _, name := range wanted {
		if !held.Has(name) {
			writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
			return false
		}
	}
	return true
}

// superuser mirrors the old /-/admin update page: system updates are
// superuser-only, never delegated through roles.
func (h *ConsoleThemes) superuser(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
	user := auth.FromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return false
	}
	if !user.IsSuperuser {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

func (h *ConsoleThemes) getList(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc, perms.ManageSite) {
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-page-not-found")))
		return
	}
	themes, err := h.deps.DB.Themes(ctx, current.ID)
	if err != nil {
		h.deps.log().Error("list themes", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	list := make(wikijson.Array, 0, len(themes))
	for _, th := range themes {
		list = append(list, wikijson.Object{
			{Key: "id", Value: th.ID},
			{Key: "name", Value: th.Name},
			{Key: "slug", Value: th.Slug},
			{Key: "active", Value: current.ThemeID != nil && *current.ThemeID == th.ID},
		})
	}

	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "themes", Value: list},
		{Key: "update", Value: h.updateInfo(ctx)},
	})
	if err != nil {
		h.deps.log().Error("marshal themes", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// updateInfo reports the last known update state. The version check itself
// runs in the background (or via the old /-/admin update page); this
// endpoint only reads the cached state, it never phones home.
func (h *ConsoleThemes) updateInfo(ctx context.Context) wikijson.Object {
	current := version.String()
	latest := ""
	if st, err := h.deps.DB.UpdateState(ctx); err == nil {
		latest = st.LatestVersion
	} else {
		h.deps.log().Error("read update state", "err", err)
	}
	return wikijson.Object{
		{Key: "current_version", Value: current},
		{Key: "latest_version", Value: latest},
		{Key: "has_update", Value: latest != "" && update.Newer(latest, current)},
	}
}

type activateThemeBody struct {
	ID int64 `json:"id"`
}

func (h *ConsoleThemes) postActivate(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	if !h.allowed(w, r, loc, perms.ManageSite) {
		return
	}

	var body activateThemeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-invalid-request")))
		return
	}
	// The theme must belong to this site; Theme returns ErrNotFound otherwise.
	if _, err := h.deps.DB.Theme(ctx, current.ID, body.ID); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-invalid-request")))
		return
	}
	settings, err := h.deps.DB.SiteSettings(ctx, current.ID)
	if err != nil {
		h.deps.log().Error("load site settings", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	next := *current
	next.ThemeID = &body.ID
	// Role-related site fields stay untouched; withRoles=false like console_site.
	if err := h.deps.DB.SaveSite(ctx, &next, settings, false); err != nil {
		h.deps.log().Error("activate theme", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	out, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal activate result", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ConsoleThemes) postCheckUpdate(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	if !h.superuser(w, r, loc) {
		return
	}
	// Deliberately read-only: a fresh network check lives behind the old
	// /-/admin update page. This endpoint surfaces the last known state so
	// the console can show it without triggering downloads or restarts.
	ver := version.String()
	latest := ""
	if st, err := h.deps.DB.UpdateState(ctx); err == nil {
		latest = st.LatestVersion
	} else {
		h.deps.log().Error("read update state", "err", err)
	}
	out, err := wikijson.Marshal(wikijson.Object{
		{Key: "ok", Value: true},
		{Key: "has_update", Value: latest != "" && update.Newer(latest, ver)},
		{Key: "latest_version", Value: latest},
	})
	if err != nil {
		h.deps.log().Error("marshal check result", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, out)
}
