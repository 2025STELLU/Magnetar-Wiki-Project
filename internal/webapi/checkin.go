package webapi

import (
	"net/http"
	"strings"

	"github.com/2025STELLU/MagnetarWiki/internal/auth"
	"github.com/2025STELLU/MagnetarWiki/internal/csrf"
	"github.com/2025STELLU/MagnetarWiki/internal/i18n"
	"github.com/2025STELLU/MagnetarWiki/internal/site"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

// CheckinPrefix is the JSON API root for the daily check-in widget.
const CheckinPrefix = "/pw-api/checkin/"

type Checkin struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*Checkin)(nil)

func NewCheckin(d Deps, next http.Handler) *Checkin {
	return &Checkin{deps: d, next: next}
}

func (h *Checkin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, CheckinPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case (head == "" || head == "status") && r.Method == http.MethodGet:
		h.status(w, r, loc)
	case rest == "" && r.Method == http.MethodPost:
		h.do(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

func (h *Checkin) currentUserID(r *http.Request) (int64, bool) {
	user := auth.FromContext(r.Context())
	if user == nil {
		return 0, false
	}
	return user.ID, true
}

func (h *Checkin) status(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	userID, ok := h.currentUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	st, err := h.deps.DB.CheckinStatus(r.Context(), userID)
	if err != nil {
		h.deps.log().Error("checkin status", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "checked_in", Value: st.CheckedIn},
		{Key: "streak", Value: st.Streak},
		{Key: "total_days", Value: st.TotalDays},
	})
	if err != nil {
		h.deps.log().Error("marshal checkin status", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Checkin) do(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	userID, ok := h.currentUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	st, xp, err := h.deps.DB.DoCheckin(ctx, userID)
	if err != nil {
		h.deps.log().Error("do checkin", "user", userID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "ok", Value: true},
		{Key: "streak", Value: st.Streak},
		{Key: "total_days", Value: st.TotalDays},
		{Key: "xp_awarded", Value: xp},
	})
	if err != nil {
		h.deps.log().Error("marshal checkin result", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}
