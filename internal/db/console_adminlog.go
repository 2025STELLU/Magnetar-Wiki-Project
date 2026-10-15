package db

import (
	"context"
	"fmt"
)

// ConsoleAdminLog lists admin log entries for the new console, with optional
// filters. actor matches the operator's username (or the stale name recorded
// when the account is gone); action is one of create/change/delete. Empty
// filters mean no filtering. Returns the page and the total matching count.
func (d *DB) ConsoleAdminLog(ctx context.Context, siteID int64, actor, action string, limit, offset int) ([]AdminNoteRow, int, error) {
	if !validAdminLogAction(action) {
		action = ""
	}
	actor = likeEscape(actor)
	rows, err := d.pool.Query(ctx, qConsoleAdminLog, actor, action, limit, offset, siteID)
	if err != nil {
		return nil, 0, fmt.Errorf("list console admin log: %w", err)
	}
	defer rows.Close()

	var out []AdminNoteRow
	for rows.Next() {
		var one AdminNoteRow
		if err := rows.Scan(&one.ID, &one.User, &one.Stale, &one.Action,
			&one.Screen, &one.Target, &one.Label, &one.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := d.pool.QueryRow(ctx, qConsoleAdminLogCount, actor, action, siteID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count console admin log: %w", err)
	}
	return out, total, nil
}

func validAdminLogAction(action string) bool {
	switch action {
	case AdminCreated, AdminChanged, AdminDeleted:
		return true
	}
	return false
}

var qConsoleAdminLog = register("ConsoleAdminLog", `
SELECT l.id, coalesce(u.username, ''), l.stale_name, l.action, l.screen, l.target, l.label, l.created_at
FROM magnetar_admin_log l
LEFT JOIN web_user u ON u.id = l.user_id
WHERE l.site_id = $5
  AND ($1 = '' OR u.username ILIKE '%' || $1 || '%' ESCAPE '\' OR l.stale_name ILIKE '%' || $1 || '%' ESCAPE '\')
  AND ($2 = '' OR l.action = $2)
ORDER BY l.created_at DESC, l.id DESC
LIMIT $3 OFFSET $4`)

var qConsoleAdminLogCount = register("ConsoleAdminLogCount", `
SELECT COUNT(*)
FROM magnetar_admin_log l
LEFT JOIN web_user u ON u.id = l.user_id
WHERE l.site_id = $3
  AND ($1 = '' OR u.username ILIKE '%' || $1 || '%' ESCAPE '\' OR l.stale_name ILIKE '%' || $1 || '%' ESCAPE '\')
  AND ($2 = '' OR l.action = $2)`)
