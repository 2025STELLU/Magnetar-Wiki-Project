package webapi

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
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

// RolesPrefix is the JSON API root for the console role manager.
const RolesPrefix = "/pw-api/console/roles/"

type ConsoleRoles struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleRoles)(nil)

func NewConsoleRoles(d Deps, next http.Handler) *ConsoleRoles {
	return &ConsoleRoles{deps: d, next: next}
}

func (h *ConsoleRoles) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, RolesPrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	head, _, _ := strings.Cut(rest, "/")
	switch {
	case head == "list" && r.Method == http.MethodGet:
		h.list(w, r, loc)
	case head == "permissions" && r.Method == http.MethodGet:
		h.permissionCatalog(w, r, loc)
	case head == "save" && r.Method == http.MethodPost:
		h.save(w, r, loc)
	case head == "delete" && r.Method == http.MethodPost:
		h.remove(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// allowed mirrors Console.allowed: signed in + holds ManageRoles.
func (h *ConsoleRoles) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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
	if !perms.Resolve(subject, nil).Has(perms.ManageRoles) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return false
	}
	return true
}

func (h *ConsoleRoles) checkCSRF(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) bool {
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

var levelSlugRe = regexp.MustCompile(`^level-(\d+)$`)

// levelOf returns the level number when the slug is a level role (level-N).
func levelOf(slug string) (int, bool) {
	m := levelSlugRe.FindStringSubmatch(slug)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 || n > db.MaxLevel {
		return 0, false
	}
	return n, true
}

func (h *ConsoleRoles) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc) {
		return
	}
	ctx := r.Context()
	rows, err := h.deps.DB.AdminRoles(ctx, siteID(ctx))
	if err != nil {
		h.deps.log().Error("list roles", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	grants, err := h.deps.DB.RolePermissions(ctx, ids)
	if err != nil {
		h.deps.log().Error("list role grants", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	byID := make(map[int64]*perms.Role, len(grants))
	for i := range grants {
		byID[grants[i].ID] = &grants[i]
	}
	items := make(wikijson.Array, 0, len(rows))
	for _, row := range rows {
		allow := []string{}
		if g, ok := byID[row.ID]; ok && g.Permissions != nil {
			allow = g.Permissions
		}
		level, isLevel := levelOf(row.Slug)
		items = append(items, wikijson.Object{
			{Key: "id", Value: row.ID},
			{Key: "slug", Value: row.Slug},
			{Key: "name", Value: row.Name},
			{Key: "description", Value: row.ShortName},
			{Key: "is_level_role", Value: isLevel},
			{Key: "level", Value: level},
			{Key: "permissions", Value: allow},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "roles", Value: items}})
	if err != nil {
		h.deps.log().Error("marshal roles", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *ConsoleRoles) permissionCatalog(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc) {
		return
	}
	items := make(wikijson.Array, 0, 48)
	for _, group := range perms.Catalog {
		for _, name := range group.Names {
			desc, ok := permissionDescriptions[name]
			if !ok {
				desc = name
			}
			items = append(items, wikijson.Object{
				{Key: "key", Value: name},
				{Key: "description", Value: desc},
				{Key: "group", Value: group.Key},
			})
		}
	}
	body, err := wikijson.Marshal(wikijson.Object{{Key: "permissions", Value: items}})
	if err != nil {
		h.deps.log().Error("marshal permission catalog", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

var roleSlugRe = regexp.MustCompile(`^[a-z0-9_]+$`)

type roleSaveBody struct {
	ID          *int64  `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Permissions []string `json:"permissions"`
}

func (h *ConsoleRoles) save(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	var body roleSaveBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	body.Slug = strings.TrimSpace(body.Slug)
	body.Name = strings.TrimSpace(body.Name)
	body.Description = strings.TrimSpace(body.Description)
	if !roleSlugRe.MatchString(body.Slug) {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	// Only catalogued permissions are accepted; anything else is dropped.
	known := make(map[string]bool, 48)
	for _, group := range perms.Catalog {
		for _, name := range group.Names {
			known[name] = true
		}
	}
	allow := make([]string, 0, len(body.Permissions))
	for _, name := range body.Permissions {
		if known[name] {
			allow = append(allow, name)
		}
	}

	ctx := r.Context()
	site := siteID(ctx)
	var row db.RoleRow
	if body.ID != nil && *body.ID != 0 {
		row, err = h.deps.DB.AdminRole(ctx, site, *body.ID)
		if err != nil {
			h.deps.log().Error("read role", "id", *body.ID, "err", err)
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
			return
		}
		// Level roles keep their slug: renaming one would break level grants.
		if _, isLevel := levelOf(row.Slug); isLevel {
			body.Slug = row.Slug
		}
	} else {
		row = db.RoleRow{InlineVisualMode: "hidden", ProfileVisualMode: "hidden"}
	}
	row.Slug = body.Slug
	row.Name = body.Name
	row.ShortName = body.Description
	row.Allow = allow
	// Deny list is managed by the old admin; the console matrix edits allows.
	if _, err := h.deps.DB.SaveRole(ctx, site, row, true); err != nil {
		h.deps.log().Error("save role", "slug", body.Slug, "err", err)
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	okBody, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal save role", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, okBody)
}

type roleDeleteBody struct {
	ID int64 `json:"id"`
}

func (h *ConsoleRoles) remove(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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
	var body roleDeleteBody
	if err := json.Unmarshal(raw, &body); err != nil || body.ID == 0 {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	ctx := r.Context()
	row, err := h.deps.DB.AdminRole(ctx, siteID(ctx), body.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if _, isLevel := levelOf(row.Slug); isLevel {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	if err := h.deps.DB.DeleteRole(ctx, siteID(ctx), body.ID); err != nil {
		h.deps.log().Error("delete role", "id", body.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	okBody, err := wikijson.Marshal(wikijson.Object{{Key: "ok", Value: true}})
	if err != nil {
		h.deps.log().Error("marshal delete role", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, okBody)
}

// permissionDescriptions is the Chinese label for each permission codename.
// The frontend renders these directly.
var permissionDescriptions = map[string]string{
	"view_articles":                "查看文章",
	"rate_articles":                "给文章评分",
	"create_articles":              "创建文章",
	"edit_articles":                "编辑文章",
	"tag_articles":                 "给文章打标签",
	"move_articles":                "移动文章",
	"lock_articles":                "锁定文章",
	"manage_article_files":         "管理文章附件",
	"delete_articles":              "删除文章",
	"reset_article_votes":          "重置文章评分",
	"comment_articles":             "发表评论",
	"view_article_comments":        "查看评论",
	"manage_article_authors":       "管理文章作者",
	"view_forum_posts":             "查看论坛帖子",
	"create_forum_posts":           "发表回帖",
	"edit_forum_posts":             "编辑帖子",
	"delete_forum_posts":           "删除帖子",
	"view_forum_threads":           "查看主题",
	"create_forum_threads":         "发表主题",
	"edit_forum_threads":           "编辑主题",
	"pin_forum_threads":            "置顶主题",
	"lock_forum_threads":           "锁定主题",
	"move_forum_threads":           "移动主题",
	"view_forum_sections":          "查看版块",
	"view_hidden_forum_sections":   "查看隐藏版块",
	"view_forum_categories":        "查看论坛分类",
	"send_direct_message":          "发送私信",
	"view_user_reports":            "查看举报",
	"view_user_tickets":            "查看工单",
	"review_membership_applications": "审核入站申请",
	"ban_members":                  "封禁成员",
	"mute_members":                 "禁言成员",
	"restrict_member_editing":      "限制成员编辑",
	"restrict_member_rating":       "限制成员评分",
	"reset_member_votes":           "重置成员投票",
	"invite_members":               "邀请成员",
	"manage_bots":                  "管理机器人",
	"manage_users":                 "管理用户",
	"manage_roles":                 "管理角色",
	"manage_site":                  "管理站点",
	"view_actions_log":             "查看管理日志",
	"manage_categories":            "管理分类",
	"manage_tags":                  "管理标签",
	"manage_forum":                 "管理论坛",
	"view_sensitive_info":          "查看敏感信息",
	"view_votes_timestamp":         "查看投票时间",
	"manage_updates":               "管理系统更新",
	"manage_permissions":           "分配权限",
}
