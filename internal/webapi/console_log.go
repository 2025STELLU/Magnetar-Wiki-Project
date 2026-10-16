package webapi

import (
	"context"
	"strconv"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/auth"
	"github.com/2025STELLU/MagnetarWiki/internal/db"
)

// logAdminAction records a console write operation in the admin action log,
// mirroring admin.Handler.noteID. Every console mutation should call this so
// the actions show up on the admin-log page.
func logAdminAction(ctx context.Context, d Deps, action, screen string, id int64, label string) {
	entry := db.AdminNote{
		Action: action,
		Screen: screen,
		Target: strconv.FormatInt(id, 10),
		Label:  label,
		SiteID: siteID(ctx),
		At:     time.Now().UTC(),
	}
	if by := auth.FromContext(ctx); by != nil {
		entry.UserID = &by.ID
		entry.Name = by.Username
	}
	if err := d.DB.WriteAdminNote(ctx, entry); err != nil {
		d.log().Error("record admin action", "screen", screen, "action", action, "err", err)
	}
}
