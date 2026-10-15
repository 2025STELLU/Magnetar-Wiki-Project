package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AdminForumPostRow is one row of the console forum-post listing.
type AdminForumPostRow struct {
	ID        int64
	Name      string
	Author    string
	Thread    string
	Category  string
	Locked    bool
	CreatedAt time.Time
}

// AdminForumPosts lists forum posts for the console, newest first, with
// optional text search (post/thread/author name) and category filter.
func (d *DB) AdminForumPosts(ctx context.Context, siteID int64, query string, categoryID int64, limit, offset int) ([]AdminForumPostRow, int, error) {
	var conds []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	conds = append(conds, "t.site_id = "+arg(siteID))
	if query != "" {
		like := "%" + likeEscape(query) + "%"
		conds = append(conds, "(p.name ILIKE "+arg(like)+" ESCAPE '\\' OR t.name ILIKE "+arg(like)+" ESCAPE '\\' OR u.username ILIKE "+arg(like)+" ESCAPE '\\')")
	}
	if categoryID > 0 {
		conds = append(conds, "t.category_id = "+arg(categoryID))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	countSQL := `SELECT count(*) FROM web_forumpost p
		JOIN web_forumthread t ON t.id = p.thread_id
		LEFT JOIN web_user u ON u.id = p.author_id ` + where
	if err := d.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count forum posts: %w", err)
	}

	listSQL := `SELECT p.id, p.name, COALESCE(u.username, ''), t.name,
		COALESCE(c.name, ''), t.is_locked, p.created_at
		FROM web_forumpost p
		JOIN web_forumthread t ON t.id = p.thread_id
		LEFT JOIN web_user u ON u.id = p.author_id
		LEFT JOIN web_forumcategory c ON c.id = t.category_id ` + where + `
		ORDER BY p.created_at DESC
		OFFSET ` + arg(offset) + ` LIMIT ` + arg(limit)
	rows, err := d.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query forum posts: %w", err)
	}
	defer rows.Close()

	var out []AdminForumPostRow
	for rows.Next() {
		var r AdminForumPostRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Author, &r.Thread, &r.Category, &r.Locked, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan forum post: %w", err)
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// AdminForumPostDetail is the console detail view of one post.
type AdminForumPostDetail struct {
	ID        int64
	Name      string
	Author    string
	Thread    string
	ThreadID  int64
	Category  string
	Locked    bool
	Summary   string
	PostCount int
	CreatedAt time.Time
}

func postSummary(source string) string {
	const maxRunes = 300
	runes := []rune(strings.TrimSpace(source))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "…"
}

// AdminForumPostDetail fetches one post with its thread, author, latest
// source excerpt and the thread's post count.
func (d *DB) AdminForumPostDetail(ctx context.Context, siteID, id int64) (*AdminForumPostDetail, error) {
	var p AdminForumPostDetail
	err := d.pool.QueryRow(ctx, `
		SELECT p.id, p.name, COALESCE(u.username, ''), t.name, t.id,
			COALESCE(c.name, ''), t.is_locked, p.created_at
		FROM web_forumpost p
		JOIN web_forumthread t ON t.id = p.thread_id
		LEFT JOIN web_user u ON u.id = p.author_id
		LEFT JOIN web_forumcategory c ON c.id = t.category_id
		WHERE p.id = $1 AND t.site_id = $2`, id, siteID).
		Scan(&p.ID, &p.Name, &p.Author, &p.Thread, &p.ThreadID,
			&p.Category, &p.Locked, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query forum post %d: %w", id, err)
	}
	contents, err := d.ForumPostContents(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if c, ok := contents[id]; ok {
		p.Summary = postSummary(c.Source)
	}
	err = d.pool.QueryRow(ctx,
		`SELECT count(*) FROM web_forumpost WHERE thread_id = $1`, p.ThreadID).Scan(&p.PostCount)
	if err != nil {
		return nil, fmt.Errorf("count thread posts: %w", err)
	}
	return &p, nil
}

// SetForumThreadLocked locks or unlocks a thread. Locking is a thread-level
// operation; the console post screen exposes it per post for convenience.
func (d *DB) SetForumThreadLocked(ctx context.Context, siteID, threadID int64, locked bool) error {
	tag, err := d.pool.Exec(ctx,
		`UPDATE web_forumthread SET is_locked = $1, updated_at = now()
		 WHERE id = $2 AND site_id = $3`, locked, threadID, siteID)
	if err != nil {
		return fmt.Errorf("set thread lock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ForumPostThreadID resolves a post to its thread, scoped to the site.
func (d *DB) ForumPostThreadID(ctx context.Context, siteID, postID int64) (int64, error) {
	var threadID int64
	err := d.pool.QueryRow(ctx,
		`SELECT p.thread_id FROM web_forumpost p
		 JOIN web_forumthread t ON t.id = p.thread_id
		 WHERE p.id = $1 AND t.site_id = $2`, postID, siteID).Scan(&threadID)
	if err != nil {
		return 0, fmt.Errorf("resolve post thread: %w", err)
	}
	return threadID, nil
}
