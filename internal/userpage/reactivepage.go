package userpage

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/auth"
	"github.com/2025STELLU/MagnetarWiki/internal/db"
	"github.com/2025STELLU/MagnetarWiki/internal/pageconfig"
	"github.com/2025STELLU/MagnetarWiki/internal/perms"
	"github.com/2025STELLU/MagnetarWiki/internal/repo"
	"github.com/2025STELLU/MagnetarWiki/internal/shell"
	"github.com/2025STELLU/MagnetarWiki/internal/site"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

const (
	FavouritesPrefix = "/-/favourites"
	RatingsPrefix    = "/-/ratings"
	LikedPostsPrefix = "/-/liked-posts"
	MyTicketsPrefix  = "/-/my-tickets"

	NotificationsPrefix = "/-/notifications"
	MessagesPrefix      = "/-/messages"

	// ConsolePrefix serves the React admin shell. The old /-/admin pages
	// stay untouched as a fallback; console subpaths are routed client-side.
	ConsolePrefix = "/-/console"

	NotificationsSubPrefix = NotificationsPrefix + "/"
	MessagesSubPrefix      = MessagesPrefix + "/"
	MyTicketSubPrefix      = MyTicketsPrefix + "/"
)

var ticketKinds = []string{"ticket", "membershipapply", "report"}

var reactivePaths = []string{
	FavouritesPrefix,
	RatingsPrefix,
	LikedPostsPrefix,
	MyTicketsPrefix,
	NotificationsPrefix,
	NotificationsPrefix + "/all",
	NotificationsPrefix + "/unread",
	MessagesPrefix,
}

func answers(path string) bool {
	if slices.Contains(reactivePaths, path) {
		return true
	}
	if path == ConsolePrefix || strings.HasPrefix(path, ConsolePrefix+"/") {
		return true
	}
	if rest, ok := strings.CutPrefix(path, MyTicketSubPrefix); ok {
		kind, raw, ok := strings.Cut(rest, "/")
		if !ok || !slices.Contains(ticketKinds, kind) {
			return false
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		return err == nil && id > 0
	}
	rest, ok := strings.CutPrefix(path, MessagesSubPrefix)
	if !ok || rest == "" {
		return false
	}
	_, err := strconv.ParseInt(rest, 10, 64)
	return err == nil
}

// consolePerms gates the /-/console shell itself. The JSON endpoints behind
// it still enforce their own fine-grained checks.
var consolePerms = []string{
	perms.ManageSite,
	perms.ManageRoles,
	perms.ManageUsers,
	perms.BanMembers,
	perms.ViewActionsLog,
}

func isConsolePath(path string) bool {
	return path == ConsolePrefix || strings.HasPrefix(path, ConsolePrefix+"/")
}

func canConsole(ctx context.Context, d *db.DB, viewer *db.User) bool {
	subject, err := repo.NewPerms(ctx, d).Subject(viewer, time.Now())
	if err != nil {
		return false
	}
	held := perms.Resolve(subject, nil)
	for _, name := range consolePerms {
		if held.Has(name) {
			return true
		}
	}
	return false
}

type ReactiveHandler struct {
	deps Deps
}

var _ http.Handler = (*ReactiveHandler)(nil)

func NewReactive(d Deps) *ReactiveHandler { return &ReactiveHandler{deps: d} }

func (h *ReactiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if !answers(r.URL.Path) {
		loc := h.deps.Bundle.For(r.Context())
		shell.New(loc, h.deps.Assets).Missing(w, r, loc.T("system.page-not-found"))
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	viewer := auth.FromContext(ctx)
	if current == nil || viewer == nil {
		redirect(w, loginPath+"?to="+url.QueryEscape(r.URL.Path), http.StatusFound)
		return
	}
	if isConsolePath(r.URL.Path) && !canConsole(ctx, h.deps.DB, viewer) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	body, err := h.page(r, current, viewer)
	if err != nil {
		h.deps.logger().Error("render reactive page", "path", r.URL.Path, "user", viewer.ID, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(body))
	}
}

func (h *ReactiveHandler) page(r *http.Request, current *db.Site, viewer *db.User) (string, error) {
	ctx := r.Context()
	loc := h.deps.Bundle.For(ctx)

	theme, err := site.ThemeURLByID(ctx, h.deps.DB, current.SystemThemeID)
	if err != nil {
		return "", err
	}
	config, err := h.config(r, viewer)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	err = shell.New(loc, h.deps.Assets).Reactive(&out, shell.Reactive{ThemeURL: theme, Config: config})
	return out.String(), err
}

func (h *ReactiveHandler) config(r *http.Request, viewer *db.User) (string, error) {
	ctx := r.Context()
	userRoles, err := h.deps.DB.RolesByUser(ctx, siteID(ctx), viewer.ID)
	if err != nil {
		return "", err
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(viewer, time.Now())
	if err != nil {
		return "", err
	}
	editor := perms.Resolve(subject, nil).Has(perms.EditArticles)
	user := pageconfig.SignedInUserJSON(viewer, userRoles, true, editor)
	reviews, err := repo.ReviewsTickets(ctx, h.deps.DB, viewer, time.Now())
	if err != nil {
		return "", err
	}
	if !reviews {
		submitted, err := h.deps.DB.OwnTicketCount(ctx, viewer.ID)
		if err != nil {
			return "", err
		}
		reviews = submitted > 0
	}
	out := wikijson.Object{{Key: "user", Value: user.Object()}, {Key: "reviewsTickets", Value: reviews}}
	if current := site.FromContext(ctx); current != nil {
		out = append(out, wikijson.Field{Key: "site", Value: wikijson.Object{
			{Key: "title", Value: current.Title},
			{Key: "systemIcon", Value: authIcon(current)},
		}})
	}
	return wikijson.Marshal(out)
}
