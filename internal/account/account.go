// Package account answers the pages a visitor signs in, signs up and signs out on.
package account

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/db"
	"github.com/2025STELLU/MagnetarWiki/internal/i18n"
	"github.com/2025STELLU/MagnetarWiki/internal/mail"
	"github.com/2025STELLU/MagnetarWiki/internal/proxyheader"
	"github.com/2025STELLU/MagnetarWiki/internal/renderer"
	"github.com/2025STELLU/MagnetarWiki/internal/roles"
	"github.com/2025STELLU/MagnetarWiki/internal/session"
	"github.com/2025STELLU/MagnetarWiki/internal/shell"
	"github.com/2025STELLU/MagnetarWiki/internal/site"
	"github.com/2025STELLU/MagnetarWiki/internal/static"
	"github.com/2025STELLU/MagnetarWiki/internal/token"
)

const (
	LoginPath  = "/-/login"
	LogoutPath = "/-/logout"

	homePath = "/"
)

type Deps struct {
	DB       *db.DB
	Sessions *session.Store
	Tokens   token.Generator
	Verifier Verifier
	Mail     mail.Sender
	Engine   renderer.Renderer
	Icons    roles.IconLoader
	Bundle   *i18n.Bundle
	Assets   *static.Assets
	Trust    *proxyheader.Trust
	Log      *slog.Logger
}

func (d Deps) clientIP(r *http.Request) *netip.Addr {
	if d.Trust == nil {
		return nil
	}
	if addr, ok := d.Trust.ClientIP(r); ok {
		return &addr
	}
	return nil
}

func (d Deps) logger() *slog.Logger {
	if d.Log == nil {
		return slog.Default()
	}
	return d.Log
}

func redirect(w http.ResponseWriter, location string, status int) {
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", "0")
	w.WriteHeader(status)
}

func destination(r *http.Request) string {
	to := r.URL.Query().Get("to")
	if to == "" || !strings.HasPrefix(to, "/") || strings.HasPrefix(to, "//") {
		return homePath
	}
	return to
}

func (d Deps) page(r *http.Request, loc *i18n.Localizer, current *db.Site, title, content string) (string, error) {
	theme, err := site.ThemeURLByID(r.Context(), d.DB, current.SystemThemeID)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	err = shell.New(loc, d.Assets).SystemPage(&out, shell.System{
		Title:     title,
		SiteTitle: current.Title,
		ThemeURL:  theme,
		BodyClass: "wikit-page",
		Content:   content,
	})
	return out.String(), err
}

func (d Deps) welcome(ctx context.Context, userID int64) error {
	return d.DB.SendNotification(ctx, 0, db.NotifyWelcome, "{}", []int64{userID}, time.Now())
}

func authIcon(s *db.Site) string {
	if s == nil || s.AuthIcon == "" {
		return ""
	}
	return "/local--files/" + s.AuthIcon
}

func siteID(ctx context.Context) int64 {
	if current := site.FromContext(ctx); current != nil {
		return current.ID
	}
	return 0
}
