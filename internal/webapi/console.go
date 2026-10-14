package webapi

import (
	"encoding/json"
	"fmt"
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

// ConsolePrefix is the JSON API root for the /-/console admin shell.
// The old /-/admin pages stay untouched as a fallback.
const ConsolePrefix = "/pw-api/console/"

type Console struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*Console)(nil)

func NewConsole(d Deps, next http.Handler) *Console {
	return &Console{deps: d, next: next}
}

func (h *Console) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, ConsolePrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case head == "overview" && rest == "overview" && r.Method == http.MethodGet:
		h.overview(w, r, loc)
	case head == "levels" && rest == "levels" && r.Method == http.MethodGet:
		h.levels(w, r, loc)
	case head == "levels" && rest == "levels" && r.Method == http.MethodPost:
		h.saveLevels(w, r, loc)
	case head == "users" && rest == "users/search" && r.Method == http.MethodGet:
		h.searchUsers(w, r, loc)
	case head == "users" && rest == "users/style" && r.Method == http.MethodGet:
		h.getUserStyle(w, r, loc)
	case head == "users" && rest == "users/style" && r.Method == http.MethodPost:
		h.setUserStyle(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// allowed mirrors Admin.allowed: the caller must be signed in and hold every
// named permission, otherwise a JSON error is written and the call stops.
func (h *Console) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, wanted ...string) bool {
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

func (h *Console) overview(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc, perms.ManageSite) {
		return
	}
	stats, err := h.deps.DB.ConsoleOverview(r.Context(), siteID(r.Context()))
	if err != nil {
		h.deps.log().Error("console overview", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "users", Value: stats.Users},
		{Key: "articles", Value: stats.Articles},
		{Key: "pending_reports", Value: stats.PendingReports},
		{Key: "pending_tickets", Value: stats.PendingTickets},
	})
	if err != nil {
		h.deps.log().Error("marshal console overview", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Console) levels(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc, perms.ManageRoles) {
		return
	}
	ctx := r.Context()
	site := siteID(ctx)
	thresholds, err := h.deps.DB.LevelThresholds(ctx, site)
	if err != nil {
		h.deps.log().Error("console level thresholds", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	prefix, err := h.deps.DB.LevelPrefix(ctx, site)
	if err != nil {
		h.deps.log().Error("console level prefix", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	out := make(wikijson.Array, 0, db.MaxLevel)
	for _, xp := range thresholds {
		out = append(out, xp)
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "thresholds", Value: out},
		{Key: "prefix", Value: prefix},
	})
	if err != nil {
		h.deps.log().Error("marshal console levels", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type levelConfigBody struct {
	Thresholds []int64 `json:"thresholds"`
	Prefix     string  `json:"prefix"`
}

func (h *Console) saveLevels(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	if !h.allowed(w, r, loc, perms.ManageRoles) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body levelConfigBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if len(body.Thresholds) != db.MaxLevel {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var thresholds [db.MaxLevel]int64
	copy(thresholds[:], body.Thresholds)
	if err := h.deps.DB.SaveLevelConfig(ctx, siteID(ctx), thresholds, body.Prefix); err != nil {
		// SaveLevelConfig rejects empty prefixes and non-increasing
		// thresholds, so a failure here is a bad request, not a crash.
		h.deps.log().Error("save console level config", "err", err)
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	writeJSON(w, http.StatusOK, field("ok", "true"))
}

// searchUsers finds users by a name substring for the console user manager.
func (h *Console) searchUsers(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc, perms.ManageUsers) {
		return
	}
	q := r.URL.Query().Get("q")
	if len([]rune(q)) < 2 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	users, err := h.deps.DB.SearchUsers(r.Context(), q, 20)
	if err != nil {
		h.deps.log().Error("console search users", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	out := make(wikijson.Array, 0, len(users))
	for _, u := range users {
		out = append(out, wikijson.Object{
			{Key: "id", Value: u.ID},
			{Key: "username", Value: u.Username},
			{Key: "display_name", Value: u.DisplayName},
			{Key: "level", Value: u.Level},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "users", Value: out}})
	if err != nil {
		h.deps.log().Error("marshal console user search", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// getUserStyle reads one user's admin-assigned name style.
func (h *Console) getUserStyle(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc, perms.ManageUsers) {
		return
	}
	var userID int64
	if _, err := fmt.Sscanf(r.URL.Query().Get("user_id"), "%d", &userID); err != nil || userID <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	style, err := h.deps.DB.GetNameStyle(r.Context(), userID)
	if err != nil {
		h.deps.log().Error("console get user style", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	var value any
	if style != nil {
		value = wikijson.Object{
			{Key: "font_family", Value: style.FontFamily},
			{Key: "color", Value: style.Color},
			{Key: "gradient_from", Value: style.GradientFrom},
			{Key: "gradient_to", Value: style.GradientTo},
			{Key: "gradient_direction", Value: style.GradientDirection},
		}
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "style", Value: value}})
	if err != nil {
		h.deps.log().Error("marshal console user style", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type userStyleBody struct {
	UserID            int64  `json:"user_id"`
	FontFamily        string `json:"font_family"`
	Color             string `json:"color"`
	GradientFrom      string `json:"gradient_from"`
	GradientTo        string `json:"gradient_to"`
	GradientDirection string `json:"gradient_direction"`
}

// setUserStyle stores an admin-assigned name style. An all-empty style
// clears it.
func (h *Console) setUserStyle(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	if !h.allowed(w, r, loc, perms.ManageUsers) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body userStyleBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if body.UserID <= 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	style := db.NameStyle{
		FontFamily:        body.FontFamily,
		Color:             body.Color,
		GradientFrom:      body.GradientFrom,
		GradientTo:        body.GradientTo,
		GradientDirection: body.GradientDirection,
	}
	if err := db.ValidateNameStyle(db.NameStyle{
		FontFamily: db.SanitizeFontFamily(style.FontFamily),
		Color:      style.Color, GradientFrom: style.GradientFrom,
		GradientTo: style.GradientTo, GradientDirection: style.GradientDirection,
	}); err != nil {
		h.deps.log().Error("console set user style: invalid", "err", err)
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var updatedBy int64
	if user := auth.FromContext(ctx); user != nil {
		updatedBy = user.ID
	}
	if err := h.deps.DB.SetNameStyle(ctx, body.UserID, style, updatedBy); err != nil {
		h.deps.log().Error("console set user style", "err", err)
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	bodyOut, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal console set user style", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, bodyOut)
}
