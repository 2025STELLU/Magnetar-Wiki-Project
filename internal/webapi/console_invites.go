package webapi

import (
	"encoding/base64"
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
	"github.com/2025STELLU/MagnetarWiki/internal/token"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

// InvitesPrefix is the JSON API root for the console invite-link manager and
// the superuser-only suspicious-address list.
const InvitesPrefix = "/pw-api/console/invites/"

type ConsoleInvites struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleInvites)(nil)

func NewConsoleInvites(d Deps, next http.Handler) *ConsoleInvites {
	return &ConsoleInvites{deps: d, next: next}
}

func (h *ConsoleInvites) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, InvitesPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case head == "list" && r.Method == http.MethodGet:
		h.list(w, r, loc)
	case head == "create" && r.Method == http.MethodPost:
		h.create(w, r, loc)
	case head == "revoke" && r.Method == http.MethodPost:
		h.revoke(w, r, loc)
	case head == "suspicious" && r.Method == http.MethodGet:
		h.suspicious(w, r, loc)
	case head == "suspicious" && r.Method == http.MethodPost:
		h.clearSuspicious(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// allowed mirrors Console.allowed: signed in + holds ManageUsers.
func (h *ConsoleInvites) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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
	if !perms.Resolve(subject, nil).Has(perms.ManageUsers) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

// superuser mirrors the old admin's suspicious screen: superusers only.
func (h *ConsoleInvites) superuser(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
	user := auth.FromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return false
	}
	if !user.IsSuperuser || !user.ActiveAt(time.Now()) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

func (h *ConsoleInvites) checkCSRF(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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

func (h *ConsoleInvites) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc) {
		return
	}
	ctx := r.Context()
	rows, _, err := h.deps.DB.AdminInvites(ctx, siteID(ctx), 100, 0)
	if err != nil {
		h.deps.log().Error("list invites", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	current := site.FromContext(ctx)
	base := ""
	if current != nil {
		base = h.deps.Trust.Scheme(r) + "://" + current.Domain
	}
	items := make(wikijson.Array, 0, len(rows))
	for _, row := range rows {
		var activatedAt any
		if row.ActivatedAt != nil {
			activatedAt = row.ActivatedAt.UTC().Format(time.RFC3339)
		}
		items = append(items, wikijson.Object{
			{Key: "id", Value: row.ID},
			{Key: "code", Value: row.Token},
			{Key: "url", Value: base + acceptPrefix + row.UIDB64 + "/" + row.Token},
			{Key: "email", Value: row.Email},
			{Key: "kind", Value: row.Kind},
			{Key: "created_at", Value: row.CreatedAt.UTC().Format(time.RFC3339)},
			{Key: "activated_at", Value: activatedAt},
			{Key: "activated_by", Value: row.ActivatedUsername},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "invites", Value: items}})
	if err != nil {
		h.deps.log().Error("marshal invites", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type inviteCreateBody struct {
	Email string `json:"email"`
}

// create mints a single-use registration invite link for the given email,
// following the same flow as the old admin's invite-link action.
func (h *ConsoleInvites) create(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	var body inviteCreateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	email := strings.TrimSpace(body.Email)
	if email == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-missing-email")))
		return
	}
	ctx := r.Context()
	if _, err := h.deps.DB.UserByEmail(ctx, email); err == nil {
		writeJSON(w, http.StatusConflict, field("error", loc.T("api-email-taken")))
		return
	} else if !errors.Is(err, db.ErrNotFound) {
		h.deps.log().Error("look up email", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	current := site.FromContext(ctx)
	if current == nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-page-not-found")))
		return
	}
	now := time.Now()
	id, err := h.deps.DB.CreateInvitedUser(ctx, email, now)
	if err != nil {
		h.deps.log().Error("create invited user", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	minted := h.deps.Tokens.Make(token.InviteValue(id, false), now)
	uid := base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
	by := auth.FromContext(ctx)
	var owner *int64
	if by != nil {
		owner = &by.ID
	}
	linkID, err := h.deps.DB.CreateInviteLink(ctx, siteID(ctx), "register", "link",
		email, "", minted, uid, owner, id, now)
	if err != nil {
		h.deps.log().Error("write invite link", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(ctx, h.deps, db.AdminCreated, "invite-links", linkID, email)
	url := h.deps.Trust.Scheme(r) + "://" + current.Domain + acceptPrefix + uid + "/" + minted
	out, err := wikijson.Marshal(wikijson.Object{
		{Key: "ok", Value: true},
		{Key: "url", Value: url},
	})
	if err != nil {
		h.deps.log().Error("marshal invite", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type inviteRevokeBody struct {
	ID int64 `json:"id"`
}

func (h *ConsoleInvites) revoke(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	var body inviteRevokeBody
	if err := json.Unmarshal(raw, &body); err != nil || body.ID == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if err := h.deps.DB.DeleteInvite(r.Context(), siteID(r.Context()), body.ID); err != nil {
		h.deps.log().Error("revoke invite", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(r.Context(), h.deps, db.AdminDeleted, "invite-links", body.ID, "")
	okBody, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal revoke invite", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, okBody)
}

func (h *ConsoleInvites) suspicious(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.superuser(w, r, loc) {
		return
	}
	found, err := h.deps.DB.SuspiciousUsers(r.Context())
	if err != nil {
		h.deps.log().Error("list suspicious users", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	items := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		var ip any
		if one.IP != nil {
			ip = *one.IP
		}
		items = append(items, wikijson.Object{
			{Key: "id", Value: one.UserID},
			{Key: "username", Value: one.Username},
			{Key: "ip", Value: ip},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "users", Value: items}})
	if err != nil {
		h.deps.log().Error("marshal suspicious users", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type suspiciousClearBody struct {
	UserID *int64 `json:"user_id"`
}

// clearSuspicious wipes tracked IP addresses, optionally for one user only,
// mirroring the old admin's clear-addresses action.
func (h *ConsoleInvites) clearSuspicious(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.checkCSRF(w, r, loc) {
		return
	}
	if !h.superuser(w, r, loc) {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var body suspiciousClearBody
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
	}
	cleared, err := h.deps.DB.ClearAddresses(r.Context(), body.UserID)
	if err != nil {
		h.deps.log().Error("clear addresses", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(r.Context(), h.deps, db.AdminDeleted, "suspicious", 0, strconv.FormatInt(cleared, 10))
	out, err := wikijson.Marshal(wikijson.Object{
		{Key: "ok", Value: true},
		{Key: "cleared", Value: cleared},
	})
	if err != nil {
		h.deps.log().Error("marshal clear addresses", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, out)
}
