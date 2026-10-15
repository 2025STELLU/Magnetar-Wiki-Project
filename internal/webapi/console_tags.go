package webapi

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
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

// TagsPrefix is the JSON API root for the console tag manager.
const TagsPrefix = "/pw-api/console/tags/"

type ConsoleTags struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleTags)(nil)

func NewConsoleTags(d Deps, next http.Handler) *ConsoleTags {
	return &ConsoleTags{deps: d, next: next}
}

func (h *ConsoleTags) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, TagsPrefix)
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
		h.saveTag(w, r, loc)
	case head == "delete" && r.Method == http.MethodPost:
		h.deleteTag(w, r, loc)
	case head == "save_category" && r.Method == http.MethodPost:
		h.saveCategory(w, r, loc)
	case head == "delete_category" && r.Method == http.MethodPost:
		h.deleteCategory(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// allowed mirrors Console.allowed: signed in + holds ManageTags.
func (h *ConsoleTags) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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
	if !perms.Resolve(subject, nil).Has(perms.ManageTags) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

func (h *ConsoleTags) checkCSRF(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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

func (h *ConsoleTags) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc) {
		return
	}
	ctx := r.Context()
	site := siteID(ctx)

	tags, _, err := h.deps.DB.AdminTags(ctx, site, "", 5000, 0)
	if err != nil {
		h.deps.log().Error("list tags", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	categories, err := h.deps.DB.AdminTagCategories(ctx, site)
	if err != nil {
		h.deps.log().Error("list tag categories", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	tagItems := make(wikijson.Array, 0, len(tags))
	for _, t := range tags {
		var categoryID any
		if t.CategoryID != nil {
			categoryID = *t.CategoryID
		}
		tagItems = append(tagItems, wikijson.Object{
			{Key: "id", Value: t.ID},
			{Key: "name", Value: t.Name},
			{Key: "category_id", Value: categoryID},
			{Key: "category_name", Value: t.CategoryName},
			{Key: "articles", Value: t.Articles},
			{Key: "is_indexed", Value: t.IsIndexed},
		})
	}
	catItems := make(wikijson.Array, 0, len(categories))
	for _, c := range categories {
		var priority any
		if c.Priority != nil {
			priority = *c.Priority
		}
		catItems = append(catItems, wikijson.Object{
			{Key: "id", Value: c.ID},
			{Key: "name", Value: c.Name},
			{Key: "slug", Value: c.Slug},
			{Key: "description", Value: c.Description},
			{Key: "priority", Value: priority},
			{Key: "tags", Value: c.Tags},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "tags", Value: tagItems},
		{Key: "categories", Value: catItems},
	})
	if err != nil {
		h.deps.log().Error("marshal tags", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

type tagSaveBody struct {
	ID         *int64 `json:"id"`
	Name       string `json:"name"`
	CategoryID *int64 `json:"category_id"`
	IsIndexed  *bool  `json:"is_indexed"`
}

func (h *ConsoleTags) saveTag(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	var body tagSaveBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}

	ctx := r.Context()
	site := siteID(ctx)
	var row db.TagRow
	if body.ID != nil && *body.ID != 0 {
		row, err = h.deps.DB.AdminTag(ctx, site, *body.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
	} else {
		row.IsIndexed = true
	}
	// A category id of 0 or missing means "no category".
	var categoryID *int64
	if body.CategoryID != nil && *body.CategoryID != 0 {
		if _, err := h.deps.DB.AdminTagCategory(ctx, site, *body.CategoryID); err != nil {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
		categoryID = body.CategoryID
	}
	row.Name = body.Name
	row.CategoryID = categoryID
	if body.IsIndexed != nil {
		row.IsIndexed = *body.IsIndexed
	}
	if err := h.deps.DB.SaveTag(ctx, site, row); err != nil {
		h.deps.log().Error("save tag", "name", body.Name, "err", err)
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	h.writeOK(w, loc)
}

type tagDeleteBody struct {
	ID int64 `json:"id"`
}

func (h *ConsoleTags) deleteTag(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	var body tagDeleteBody
	if err := json.Unmarshal(raw, &body); err != nil || body.ID == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if err := h.deps.DB.DeleteTag(r.Context(), siteID(r.Context()), body.ID); err != nil {
		h.deps.log().Error("delete tag", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	h.writeOK(w, loc)
}

var tagCategorySlugRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type tagCategorySaveBody struct {
	ID          *int64  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Priority    *int    `json:"priority"`
}

func (h *ConsoleTags) saveCategory(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	var body tagCategorySaveBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Slug = strings.TrimSpace(body.Slug)
	body.Description = strings.TrimSpace(body.Description)
	if body.Name == "" || !tagCategorySlugRe.MatchString(body.Slug) {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}

	ctx := r.Context()
	site := siteID(ctx)
	var row db.TagCategoryRow
	if body.ID != nil && *body.ID != 0 {
		row, err = h.deps.DB.AdminTagCategory(ctx, site, *body.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
	}
	row.Name = body.Name
	row.Slug = body.Slug
	row.Description = body.Description
	row.Priority = body.Priority
	if err := h.deps.DB.SaveTagCategory(ctx, site, row); err != nil {
		h.deps.log().Error("save tag category", "slug", body.Slug, "err", err)
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	h.writeOK(w, loc)
}

func (h *ConsoleTags) deleteCategory(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	var body tagDeleteBody
	if err := json.Unmarshal(raw, &body); err != nil || body.ID == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if err := h.deps.DB.DeleteTagCategory(r.Context(), siteID(r.Context()), body.ID); err != nil {
		h.deps.log().Error("delete tag category", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	h.writeOK(w, loc)
}

func (h *ConsoleTags) writeOK(w http.ResponseWriter, loc *i18n.Localizer) {
	body, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal tag ok", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}
