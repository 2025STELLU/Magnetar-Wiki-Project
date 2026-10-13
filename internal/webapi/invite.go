package webapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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

const acceptPrefix = "/-/accept/"

const (
	inviteKindRegister = "register"
	inviteByLink       = "link"
)

type inviteRequest struct {
	Email string  `json:"email"`
	Roles []int64 `json:"roles"`
}

func (h *Users) invite(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	current := site.FromContext(ctx)
	if current == nil {
		h.deps.log().Error("invite", "err", errors.New("the request carries no site"))
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		refuseCSRF(w, r, loc)
		return
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(user, time.Now())
	if err != nil {
		h.deps.log().Error("resolve permissions", "user", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if !perms.Resolve(subject, nil).Has(perms.ManageUsers) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}

	raw, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var input inviteRequest
	if json.Unmarshal(raw, &input) != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-json")))
		return
	}
	email := strings.TrimSpace(input.Email)
	if email == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-missing-email")))
		return
	}
	if _, err := h.deps.DB.UserByEmail(ctx, email); err == nil {
		writeJSON(w, http.StatusConflict, field("error", loc.T("api-email-taken")))
		return
	} else if !errors.Is(err, db.ErrNotFound) {
		h.deps.log().Error("look up email", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	body, err := h.mintInvite(r, current, user, email, input.Roles)
	if err != nil {
		h.deps.log().Error("write invitation", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Users) mintInvite(r *http.Request, current *db.Site, by *db.User, email string, roles []int64) (string, error) {
	ctx := r.Context()
	now := time.Now()
	id, err := h.deps.DB.CreateInvitedUser(ctx, email, now)
	if err != nil {
		return "", err
	}
	for _, role := range roles {
		if err := h.deps.DB.GrantRole(ctx, siteID(ctx), id, role); err != nil {
			return "", err
		}
	}
	minted := h.deps.Tokens.Make(token.InviteValue(id, false), now)
	uid := base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
	link := h.deps.Trust.Scheme(r) + "://" + current.Domain + acceptPrefix + uid + "/" + minted

	owner := by.ID
	if _, err := h.deps.DB.CreateInviteLink(ctx, siteID(ctx), inviteKindRegister, inviteByLink,
		email, "", minted, uid, &owner, id, now); err != nil {
		return "", err
	}
	return wikijson.Marshal(wikijson.Object{
		{Key: "email", Value: email},
		{Key: "invitationUrl", Value: link},
		{Key: "userId", Value: id},
	})
}
