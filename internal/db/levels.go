package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// MagnetarWiki level system.
//
// Levels are 1..10 (MaxLevel). Each level maps to a per-site role
// (slug "level-N"): the role's permission set IS that level's permissions,
// edited by admins through the permission-group UI. Level-ups never touch
// the permission resolver itself.
//
// Two things are admin-customizable per site:
//   - thresholds: total XP needed to reach each level (magnetar_level_config)
//   - prefix: the label shown before the level number, e.g. "Lv" -> "Lv3",
//     stored in web_settings.level_prefix and applied to every display.
//
// XP awards live in one place so the numbers stay tunable. The ledger table
// (magnetar_xp_ledger) dedups awards: one (user, source, ref) pays out at
// most once, which is also the anti-farming mechanism. One-time actions use
// a stable ref; daily actions fold the date into the ref.

const MaxLevel = 10

// DefaultLevelThresholds is the XP total required to reach each level
// (index 0 = level 1). Gaps: 100, 200, 300, ... 900.
var DefaultLevelThresholds = [MaxLevel]int64{0, 100, 300, 600, 1000, 1500, 2100, 2800, 3600, 4500}

// XP awards per action.
const (
	XPArticleCreate  = 10
	XPArticleEdit    = 2
	XPForumThread    = 3
	XPForumPost      = 1
	XPRatingReceived = 2
	XPDailyLogin     = 1
)

// XP sources, stored in magnetar_xp_ledger.source.
const (
	XPSourceArticleCreate = "article_create"
	XPSourceArticleEdit   = "article_edit"
	XPSourceForumThread   = "forum_thread"
	XPSourceForumPost     = "forum_post"
	XPSourceRatingGiven   = "rating_received"
	XPSourceDailyLogin    = "daily_login"
)

// FormatLevel renders a level with the site's prefix, e.g. "Lv3".
// Change the prefix in settings and every display follows.
func FormatLevel(prefix string, level int) string {
	return prefix + strconv.Itoa(level)
}

var qLevelThresholds = register("LevelThresholds", `
SELECT level, xp_required FROM magnetar_level_config
WHERE site_id = $1 ORDER BY level`)

// LevelThresholds returns the per-site XP thresholds, falling back to the
// defaults when the site has no custom config yet.
func (d *DB) LevelThresholds(ctx context.Context, siteID int64) ([MaxLevel]int64, error) {
	thresholds := DefaultLevelThresholds
	rows, err := d.pool.Query(ctx, qLevelThresholds, siteID)
	if err != nil {
		return thresholds, fmt.Errorf("read level thresholds: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var level int
		var xp int64
		if err := rows.Scan(&level, &xp); err != nil {
			return thresholds, fmt.Errorf("scan level threshold: %w", err)
		}
		if level >= 1 && level <= MaxLevel {
			thresholds[level-1] = xp
		}
	}
	if err := rows.Err(); err != nil {
		return thresholds, err
	}
	return thresholds, nil
}

// LevelForXP maps an XP total to a level using the given thresholds,
// capped at MaxLevel.
func LevelForXP(xp int64, thresholds [MaxLevel]int64) int {
	level := 1
	for i := 1; i < MaxLevel; i++ {
		if xp < thresholds[i] {
			break
		}
		level = i + 1
	}
	return level
}

var qLevelPrefix = register("LevelPrefix", `
SELECT level_prefix FROM web_settings WHERE site_id = $1 AND category_id IS NULL`)

// LevelPrefix returns the site's level label, e.g. "Lv".
func (d *DB) LevelPrefix(ctx context.Context, siteID int64) (string, error) {
	var prefix string
	if err := d.pool.QueryRow(ctx, qLevelPrefix, siteID).Scan(&prefix); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "Lv", nil
		}
		return "", fmt.Errorf("read level prefix: %w", err)
	}
	if prefix == "" {
		return "Lv", nil
	}
	return prefix, nil
}

var qSaveLevelConfig = register("SaveLevelConfig", `
INSERT INTO magnetar_level_config (site_id, level, xp_required)
VALUES ($1, $2, $3)
ON CONFLICT (site_id, level) DO UPDATE SET xp_required = EXCLUDED.xp_required`)

var qSaveLevelPrefix = register("SaveLevelPrefix", `
UPDATE web_settings SET level_prefix = $2 WHERE site_id = $1 AND category_id IS NULL`)

// SaveLevelConfig stores admin-customized thresholds and prefix for a site.
func (d *DB) SaveLevelConfig(ctx context.Context, siteID int64, thresholds [MaxLevel]int64, prefix string) error {
	if prefix == "" {
		return fmt.Errorf("save level config: prefix must not be empty")
	}
	for i := 1; i < MaxLevel; i++ {
		if thresholds[i] <= thresholds[i-1] {
			return fmt.Errorf("save level config: thresholds must strictly increase")
		}
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save level config: %w", err)
	}
	defer tx.Rollback(ctx)
	for i, xp := range thresholds {
		if _, err := tx.Exec(ctx, qSaveLevelConfig, siteID, i+1, xp); err != nil {
			return fmt.Errorf("save threshold for level %d: %w", i+1, err)
		}
	}
	if _, err := tx.Exec(ctx, qSaveLevelPrefix, siteID, prefix); err != nil {
		return fmt.Errorf("save level prefix: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit level config: %w", err)
	}
	return nil
}

// seedLevelConfig writes the default thresholds for a freshly created site.
func (d *DB) seedLevelConfig(ctx context.Context, tx pgx.Tx, siteID int64) error {
	for i, xp := range DefaultLevelThresholds {
		if _, err := tx.Exec(ctx, qSaveLevelConfig, siteID, i+1, xp); err != nil {
			return fmt.Errorf("seed threshold for level %d: %w", i+1, err)
		}
	}
	return nil
}

var qLedgerAward = register("LedgerAward", `
INSERT INTO magnetar_xp_ledger (user_id, day, source, ref_id, amount)
VALUES ($1, CURRENT_DATE, $2, $3, $4)
ON CONFLICT DO NOTHING
RETURNING true`)

var qBumpXP = register("BumpXP", `
UPDATE web_user SET xp = xp + $2 WHERE id = $1
RETURNING xp, level`)

var qSetLevel = register("SetLevel", `
UPDATE web_user SET level = $2 WHERE id = $1`)

// AwardXP grants XP to a user. siteID 0 means "no site context": XP is still
// recorded but no level role is granted. It returns whether the award caused
// a level-up and the user's new level. Duplicate (source, ref) awards are
// silently ignored. Each statement is atomic and the role grant is
// idempotent, so a failed award never leaves a half-applied state and the
// underlying content write must not treat an XP error as fatal.
func (d *DB) AwardXP(ctx context.Context, siteID, userID int64, amount int, source, refID string) (leveledUp bool, newLevel int, err error) {
	if amount <= 0 {
		return false, 0, fmt.Errorf("award xp: amount must be positive, got %d", amount)
	}
	var userType string
	if err := d.pool.QueryRow(ctx, `SELECT type FROM web_user WHERE id = $1`, userID).Scan(&userType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, 0, ErrNotFound
		}
		return false, 0, fmt.Errorf("award xp: lookup user %d: %w", userID, err)
	}
	if userType != UserTypeNormal && userType != UserTypeWikidot {
		return false, 0, nil // system/bot accounts don't earn XP
	}

	var awarded bool
	if err := d.pool.QueryRow(ctx, qLedgerAward, userID, source, refID, amount).Scan(&awarded); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, 0, nil // already awarded
		}
		return false, 0, fmt.Errorf("award xp: ledger: %w", err)
	}

	var newXP int64
	var oldLevel int
	if err := d.pool.QueryRow(ctx, qBumpXP, userID, amount).Scan(&newXP, &oldLevel); err != nil {
		return false, 0, fmt.Errorf("award xp: bump: %w", err)
	}
	thresholds, err := d.LevelThresholds(ctx, siteID)
	if err != nil {
		return false, 0, err
	}
	newLevel = LevelForXP(newXP, thresholds)
	if newLevel > MaxLevel {
		newLevel = MaxLevel
	}
	leveledUp = newLevel > oldLevel
	if leveledUp {
		if _, err := d.pool.Exec(ctx, qSetLevel, userID, newLevel); err != nil {
			return false, newLevel, fmt.Errorf("award xp: set level: %w", err)
		}
	}
	if siteID != 0 {
		// Always reconcile the site role for the current level, not just on
		// level-up: XP earned without site context (e.g. daily login) can
		// otherwise leave the role behind. GrantRole is idempotent.
		roleID, err := d.ensureLevelRole(ctx, siteID, newLevel)
		if err != nil {
			return leveledUp, newLevel, fmt.Errorf("award xp: level role: %w", err)
		}
		if err := d.GrantRole(ctx, siteID, userID, roleID); err != nil {
			return leveledUp, newLevel, fmt.Errorf("award xp: grant level role: %w", err)
		}
	}
	return leveledUp, newLevel, nil
}

// awardXPQuiet grants XP without failing the caller's content write if the
// XP machinery has a problem. Gamification must never break posting.
func (d *DB) awardXPQuiet(ctx context.Context, siteID, userID int64, amount int, source, refID string) {
	_, _, _ = d.AwardXP(ctx, siteID, userID, amount, source, refID)
}

func (d *DB) siteOfArticle(ctx context.Context, articleID int64) int64 {
	var siteID int64
	if err := d.pool.QueryRow(ctx, `SELECT site_id FROM web_article WHERE id = $1`, articleID).Scan(&siteID); err != nil {
		return 0
	}
	return siteID
}

func (d *DB) siteOfThread(ctx context.Context, threadID int64) int64 {
	var siteID int64
	if err := d.pool.QueryRow(ctx, `SELECT site_id FROM web_forumthread WHERE id = $1`, threadID).Scan(&siteID); err != nil {
		return 0
	}
	return siteID
}

func (d *DB) articleAuthors(ctx context.Context, articleID int64) []int64 {
	rows, err := d.pool.Query(ctx, qReadArticleAuthors, articleID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil
		}
		ids = append(ids, id)
	}
	return ids
}

// levelRoleSlug is the per-site role slug for a level, e.g. "level-3".
func levelRoleSlug(level int) string { return fmt.Sprintf("level-%d", level) }

// ensureLevelRole returns the site's role for a level, creating it
// (permission-free) the first time it is needed. Admins attach the level's
// permissions to this role through the permission-group UI.
func (d *DB) ensureLevelRole(ctx context.Context, siteID int64, level int) (int64, error) {
	slug := levelRoleSlug(level)
	ids, err := d.RoleIDsBySlug(ctx, siteID, []string{slug})
	if err != nil {
		return 0, err
	}
	if id, ok := ids[slug]; ok {
		return id, nil
	}
	prefix, err := d.LevelPrefix(ctx, siteID)
	if err != nil {
		return 0, err
	}
	name := FormatLevel(prefix, level)
	var id int64
	err = d.pool.QueryRow(ctx, qInsertBuiltInRole, siteID, slug, 10+level, name, false, "status").Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create level role %q: %w", slug, err)
	}
	return id, nil
}
