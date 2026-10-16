package webapi

import (
	"encoding/json"
	"errors"
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

// ForumPrefix is the JSON API root for the console forum manager
// (forum sections and their categories).
const ForumPrefix = "/pw-api/console/forum/"

type ConsoleForum struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleForum)(nil)

func NewConsoleForum(d Deps, next http.Handler) *ConsoleForum {
	return &ConsoleForum{deps: d, next: next}
}

func (h *ConsoleForum) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, ForumPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case head == "list" && r.Method == http.MethodGet:
		h.list(w, r, loc)
	case head == "save_section" && r.Method == http.MethodPost:
		h.saveSection(w, r, loc)
	case head == "delete_section" && r.Method == http.MethodPost:
		h.deleteSection(w, r, loc)
	case head == "save_category" && r.Method == http.MethodPost:
		h.saveCategory(w, r, loc)
	case head == "delete_category" && r.Method == http.MethodPost:
		h.deleteCategory(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// allowed mirrors Console.allowed: signed in + holds ManageForum.
func (h *ConsoleForum) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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

func (h *ConsoleForum) checkCSRF(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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

func (h *ConsoleForum) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc) {
		return
	}
	ctx := r.Context()
	site := siteID(ctx)

	sections, err := h.deps.DB.AdminForumSections(ctx, site)
	if err != nil {
		h.deps.log().Error("list forum sections", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	categories, err := h.deps.DB.AdminForumCategories(ctx, site)
	if err != nil {
		h.deps.log().Error("list forum categories", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	// Nest categories under their section, preserving DB order.
	bySection := make(map[int64]wikijson.Array, len(sections))
	for _, c := range categories {
		bySection[c.SectionID] = append(bySection[c.SectionID], wikijson.Object{
			{Key: "id", Value: c.ID},
			{Key: "name", Value: c.Name},
			{Key: "description", Value: c.Description},
		})
	}
	sectionItems := make(wikijson.Array, 0, len(sections))
	for _, s := range sections {
		cats := bySection[s.ID]
		if cats == nil {
			cats = wikijson.Array{}
		}
		sectionItems = append(sectionItems, wikijson.Object{
			{Key: "id", Value: s.ID},
			{Key: "name", Value: s.Name},
			{Key: "description", Value: s.Description},
			{Key: "categories", Value: cats},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "sections", Value: sectionItems},
	})
	if err != nil {
		h.deps.log().Error("marshal forum sections", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type forumSectionSaveBody struct {
	ID          *int64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *ConsoleForum) saveSection(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.checkCSRF(w, r, loc) {
		return
	}
	if !h.allowed(w, r, loc) {
		return
	}
	var body forumSectionSaveBody
	if !h.readBody(w, r, loc, &body) {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Description = strings.TrimSpace(body.Description)
	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}

	ctx := r.Context()
	site := siteID(ctx)
	var row db.ForumSectionRow
	isNew := body.ID == nil || *body.ID == 0
	if !isNew {
		var err error
		row, err = h.deps.DB.AdminForumSection(ctx, site, *body.ID)
		if errors.Is(err, db.ErrNotFound) {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
		if err != nil {
			h.deps.log().Error("read forum section", "id", *body.ID, "err", err)
			writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
			return
		}
	}
	// Name/description are the console-editable fields; order and
	// visibility flags keep their stored values.
	row.Name = body.Name
	row.Description = body.Description
	if err := h.deps.DB.SaveForumSection(ctx, site, row); err != nil {
		h.deps.log().Error("save forum section", "name", body.Name, "err", err)
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	did := db.AdminChanged
	if isNew {
		did = db.AdminCreated
	}
	logAdminAction(ctx, h.deps, did, "forum-sections", row.ID, row.Name)
	h.writeOK(w, loc)
}

type forumDeleteBody struct {
	ID int64 `json:"id"`
}

func (h *ConsoleForum) deleteSection(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.checkCSRF(w, r, loc) {
		return
	}
	if !h.allowed(w, r, loc) {
		return
	}
	var body forumDeleteBody
	if !h.readBody(w, r, loc, &body) || body.ID == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	ctx := r.Context()
	row, err := h.deps.DB.AdminForumSection(ctx, siteID(ctx), body.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if err := h.deps.DB.DeleteForumSection(ctx, siteID(ctx), body.ID); err != nil {
		h.deps.log().Error("delete forum section", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(ctx, h.deps, db.AdminDeleted, "forum-sections", body.ID, row.Name)
	h.writeOK(w, loc)
}

type forumCategorySaveBody struct {
	ID          *int64 `json:"id"`
	SectionID   int64  `json:"section_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *ConsoleForum) saveCategory(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.checkCSRF(w, r, loc) {
		return
	}
	if !h.allowed(w, r, loc) {
		return
	}
	var body forumCategorySaveBody
	if !h.readBody(w, r, loc, &body) {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Description = strings.TrimSpace(body.Description)
	if body.Name == "" || body.SectionID == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}

	ctx := r.Context()
	site := siteID(ctx)
	// The target section must exist on this site.
	if _, err := h.deps.DB.AdminForumSection(ctx, site, body.SectionID); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var row db.ForumCategoryRow
	isNew := body.ID == nil || *body.ID == 0
	if !isNew {
		var err error
		row, err = h.deps.DB.AdminForumCategory(ctx, site, *body.ID)
		if errors.Is(err, db.ErrNotFound) {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
		if err != nil {
			h.deps.log().Error("read forum category", "id", *body.ID, "err", err)
			writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
			return
		}
	}
	row.SectionID = body.SectionID
	row.Name = body.Name
	row.Description = body.Description
	if err := h.deps.DB.SaveForumCategory(ctx, site, row); err != nil {
		h.deps.log().Error("save forum category", "name", body.Name, "err", err)
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	did := db.AdminChanged
	if isNew {
		did = db.AdminCreated
	}
	logAdminAction(ctx, h.deps, did, "forum-categories", row.ID, row.Name)
	h.writeOK(w, loc)
}

func (h *ConsoleForum) deleteCategory(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.checkCSRF(w, r, loc) {
		return
	}
	if !h.allowed(w, r, loc) {
		return
	}
	var body forumDeleteBody
	if !h.readBody(w, r, loc, &body) || body.ID == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	ctx := r.Context()
	row, err := h.deps.DB.AdminForumCategory(ctx, siteID(ctx), body.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if err := h.deps.DB.DeleteForumCategory(ctx, siteID(ctx), body.ID); err != nil {
		h.deps.log().Error("delete forum category", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(ctx, h.deps, db.AdminDeleted, "forum-categories", body.ID, row.Name)
	h.writeOK(w, loc)
}

func (h *ConsoleForum) readBody(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, dst any) bool {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxParsed))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return false
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return false
	}
	return true
}

func (h *ConsoleForum) writeOK(w http.ResponseWriter, loc *i18n.Localizer) {
	body, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal forum ok", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}
