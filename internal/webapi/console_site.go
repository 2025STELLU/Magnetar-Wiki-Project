package webapi

import (
	"encoding/json"
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
	"github.com/2025STELLU/MagnetarWiki/internal/timezone"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

// ConsoleSitePrefix is the JSON API root for the /-/console site settings page.
const ConsoleSitePrefix = "/pw-api/console/site/"

type ConsoleSite struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*ConsoleSite)(nil)

func NewConsoleSite(d Deps, next http.Handler) *ConsoleSite {
	return &ConsoleSite{deps: d, next: next}
}

func (h *ConsoleSite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, ConsoleSitePrefix)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	switch {
	case rest == "settings" && r.Method == http.MethodGet:
		h.getSettings(w, r, loc)
	case rest == "settings" && r.Method == http.MethodPost:
		h.saveSettings(w, r, loc)
	default:
		h.next.ServeHTTP(w, r)
	}
}

// allowed mirrors Console.allowed: signed in + holds every named permission.
func (h *ConsoleSite) allowed(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, wanted ...string) bool {
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

var (
	siteSlugPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

	siteRatingModes = []string{"disabled", "updown", "stars"}
	siteTagModes    = []string{"disabled", "enabled"}
	siteEmailPolicy = []string{db.EmailOptional, db.EmailRequired, db.EmailAtSignup}
)

// siteSettingsBody is the JSON shape the console form sends and receives.
type siteSettingsBody struct {
	Slug            string  `json:"slug"`
	Title           string  `json:"title"`
	Headline        string  `json:"headline"`
	Domain          string  `json:"domain"`
	MediaDomain     string  `json:"media_domain"`
	HomePage        string  `json:"home_page"`
	FooterLicense   string  `json:"footer_license"`
	SignupNotice    string  `json:"signup_notice"`
	PasswordHelp    string  `json:"password_help"`
	EmailPolicy     string  `json:"email_policy"`
	Language        string  `json:"language"`
	TimeZone        string  `json:"time_zone"`
	ThemeID         *int64  `json:"theme_id"`
	SystemThemeID   *int64  `json:"system_theme_id"`
	RatingMode      string  `json:"rating_mode"`
	CreateTags      string  `json:"create_tags"`
}

func (h *ConsoleSite) getSettings(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	if !h.allowed(w, r, loc, perms.ManageSite) {
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-page-not-found")))
		return
	}
	settings, err := h.deps.DB.SiteSettings(ctx, current.ID)
	if err != nil {
		h.deps.log().Error("site settings", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	themes, err := h.deps.DB.Themes(ctx, current.ID)
	if err != nil {
		h.deps.log().Error("site themes", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	themeList := make(wikijson.Array, 0, len(themes))
	for _, th := range themes {
		themeList = append(themeList, wikijson.Object{
			{Key: "id", Value: th.ID},
			{Key: "name", Value: th.Name},
			{Key: "slug", Value: th.Slug},
		})
	}
	langList := make(wikijson.Array, 0)
	for _, c := range h.deps.Bundle.Choices() {
		langList = append(langList, wikijson.Object{
			{Key: "tag", Value: c.Tag},
			{Key: "name", Value: c.Name},
		})
	}

	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "site", Value: wikijson.Object{
			{Key: "slug", Value: current.Slug},
			{Key: "title", Value: current.Title},
			{Key: "headline", Value: current.Headline},
			{Key: "domain", Value: current.Domain},
			{Key: "media_domain", Value: current.MediaDomain},
			{Key: "home_page", Value: current.HomePage},
			{Key: "footer_license", Value: current.FooterLicense},
			{Key: "signup_notice", Value: current.SignupNotice},
			{Key: "password_help", Value: current.PasswordHelp},
			{Key: "email_policy", Value: current.EmailPolicy},
			{Key: "language", Value: current.Language},
			{Key: "time_zone", Value: current.TimeZone},
			{Key: "theme_id", Value: int64OrNull(current.ThemeID)},
			{Key: "system_theme_id", Value: int64OrNull(current.SystemThemeID)},
		}},
		{Key: "settings", Value: wikijson.Object{
			{Key: "rating_mode", Value: settings.RatingMode},
			{Key: "create_tags", Value: settings.CreateTags},
		}},
		{Key: "themes", Value: themeList},
		{Key: "languages", Value: langList},
		{Key: "rating_modes", Value: siteRatingModes},
		{Key: "tag_modes", Value: siteTagModes},
		{Key: "email_policies", Value: siteEmailPolicy},
	})
	if err != nil {
		h.deps.log().Error("marshal site settings", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func int64OrNull(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func (h *ConsoleSite) saveSettings(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
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

	var body siteSettingsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-invalid-request")))
		return
	}
	if problem := checkConsoleSite(h.deps.Bundle, body); problem != "" {
		writeJSON(w, http.StatusBadRequest, field("error", problem))
		return
	}

	next := *current
	next.Slug = strings.TrimSpace(body.Slug)
	next.Title = strings.TrimSpace(body.Title)
	next.Headline = strings.TrimSpace(body.Headline)
	next.Domain = strings.TrimSpace(body.Domain)
	next.MediaDomain = strings.TrimSpace(body.MediaDomain)
	next.HomePage = strings.TrimSpace(body.HomePage)
	next.FooterLicense = body.FooterLicense
	next.SignupNotice = body.SignupNotice
	next.PasswordHelp = body.PasswordHelp
	next.EmailPolicy = body.EmailPolicy
	next.Language = body.Language
	next.TimeZone = strings.TrimSpace(body.TimeZone)
	next.ThemeID = body.ThemeID
	next.SystemThemeID = body.SystemThemeID

	settings := db.SiteSettings{
		RatingMode: body.RatingMode,
		CreateTags: body.CreateTags,
	}
	// Role-related site fields (default/verified roles, membership password)
	// stay in the old /-/admin for now; the console only touches the rest.
	if err := h.deps.DB.SaveSite(ctx, &next, settings, false); err != nil {
		h.deps.log().Error("save site settings", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	logAdminAction(ctx, h.deps, db.AdminChanged, "site", siteID(ctx), next.Title)

	resp := wikijson.Object{{Key: "ok", Value: true}}
	if next.Domain != current.Domain || next.MediaDomain != current.MediaDomain {
		resp = append(resp, wikijson.Field{Key: "warning",
			Value: "域名已修改为 " + next.Domain + "，请确认 DNS 解析已生效，否则站点可能无法访问。"})
	}
	out, err := wikijson.Marshal(resp)
	if err != nil {
		h.deps.log().Error("marshal site save result", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// checkConsoleSite mirrors the old admin's checkSite validation.
func checkConsoleSite(bundle *i18n.Bundle, b siteSettingsBody) string {
	switch {
	case !siteSlugPattern.MatchString(strings.TrimSpace(b.Slug)):
		return "站点标识格式不正确（只允许字母、数字、下划线和连字符）"
	case strings.TrimSpace(b.Title) == "":
		return "站点标题不能为空"
	case !site.ValidHost(strings.TrimSpace(b.Domain)) || !site.ValidHost(strings.TrimSpace(b.MediaDomain)):
		return "域名格式不正确"
	case strings.TrimSpace(b.HomePage) == "":
		return "首页页面名不能为空"
	case !contains(siteEmailPolicy, b.EmailPolicy):
		return "邮箱策略取值不正确"
	case !bundle.Has(b.Language):
		return "语言取值不正确"
	case !timezone.Valid(strings.TrimSpace(b.TimeZone)):
		return "时区取值不正确"
	case !contains(siteRatingModes, b.RatingMode) || !contains(siteTagModes, b.CreateTags):
		return "功能开关取值不正确"
	}
	return ""
}

func contains(list []string, want string) bool {
	for _, one := range list {
		if one == want {
			return true
		}
	}
	return false
}
