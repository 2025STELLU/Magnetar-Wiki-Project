package pageconfig

import (
	"github.com/2025STELLU/MagnetarWiki/internal/db"
	"github.com/2025STELLU/MagnetarWiki/internal/i18n"
	"github.com/2025STELLU/MagnetarWiki/internal/roles"
	"github.com/2025STELLU/MagnetarWiki/internal/wikijson"
)

const (
	typeAnonymous = "anonymous"
	typeSystem    = "system"
)

type UserJSON struct {
	Type       string
	ID         any
	Name       any
	Username   any
	URLName    any
	IsActive   bool
	Avatar     any
	ShowAvatar bool
	Admin      bool
	Staff      bool
	Editor     any
	Roles      any
	NameStyle  *db.NameStyle
}

func (u UserJSON) Object() wikijson.Object {
	return wikijson.Object{
		{Key: "type", Value: u.Type},
		{Key: "id", Value: u.ID},
		{Key: "name", Value: u.Name},
		{Key: "username", Value: u.Username},
		{Key: "urlName", Value: u.URLName},
		{Key: "isActive", Value: u.IsActive},
		{Key: "avatar", Value: u.Avatar},
		{Key: "showAvatar", Value: u.ShowAvatar},
		{Key: "admin", Value: u.Admin},
		{Key: "staff", Value: u.Staff},
		{Key: "editor", Value: u.Editor},
		{Key: "roles", Value: u.Roles},
		{Key: "name_style", Value: nameStyleJSON(u.NameStyle)},
	}
}

// nameStyleJSON renders the admin-configured username style for the React
// frontend, which expects snake_case keys. Nil stays nil.
func nameStyleJSON(s *db.NameStyle) any {
	if s == nil {
		return nil
	}
	return wikijson.Object{
		{Key: "font_family", Value: s.FontFamily},
		{Key: "color", Value: s.Color},
		{Key: "gradient_from", Value: s.GradientFrom},
		{Key: "gradient_to", Value: s.GradientTo},
		{Key: "gradient_direction", Value: s.GradientDirection},
	}
}

func (u UserJSON) JSON() (string, error) { return wikijson.Marshal(u.Object()) }

func SystemUserJSON() UserJSON {
	return UserJSON{Type: typeSystem, IsActive: true, Editor: false}
}

func AnonymousUserJSON(loc *i18n.Localizer, showAvatar bool) UserJSON {
	return UserJSON{
		Type:       typeAnonymous,
		Name:       loc.T("user-anonymous"),
		IsActive:   true,
		ShowAvatar: showAvatar,
		Editor:     false,
	}
}

func SignedInUserJSON(u *db.User, userRoles []roles.Role, showAvatar, editor bool) UserJSON {
	out := UserJSON{
		Type:       u.Type,
		ID:         u.ID,
		Name:       u.DisplayLabel(),
		Username:   u.Username,
		URLName:    u.URLName(),
		IsActive:   u.IsActive,
		ShowAvatar: showAvatar,
		Admin:      u.IsSuperuser,
		Staff:      IsStaff(u, userRoles),
		Editor:     editor,
		Roles:      visualSlugs(userRoles),
		NameStyle:  u.NameStyle,
	}
	if u.Avatar != "" {
		out.Avatar = "/local--files/" + u.Avatar
	}
	return out
}

// The superuser flag is deliberately not per-site. It belongs to whoever runs
// the server, and every other way into the admin comes from a role, which is.
func IsStaff(u *db.User, userRoles []roles.Role) bool {
	if u.IsSuperuser {
		return true
	}
	for _, role := range userRoles {
		if role.IsStaff {
			return true
		}
	}
	return false
}

func visualSlugs(userRoles []roles.Role) []string {
	out := []string{}
	for _, role := range userRoles {
		if role.IsVisual() {
			out = append(out, role.Slug)
		}
	}
	return out
}

type LoginStatus struct {
	User              *db.User
	Roles             []roles.Role
	CanEditArticles   bool
	NotificationCount int
}

func (l LoginStatus) JSON(loc *i18n.Localizer) (string, error) {
	user := AnonymousUserJSON(loc, true)
	if l.User != nil {
		user = SignedInUserJSON(l.User, l.Roles, true, l.CanEditArticles)
	}
	return wikijson.Marshal(wikijson.Object{
		{Key: "user", Value: user.Object()},
		{Key: "notificationCount", Value: l.NotificationCount},
	})
}
