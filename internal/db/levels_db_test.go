package db

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/migrate"
)

// Database integration tests for the level system. Skipped without
// MAGNETAR_TEST_DSN; the schema is migrated in-test (idempotent).

func migrateLevelTestDB(t *testing.T, dsn string) {
	t.Helper()
	if _, err := migrate.Run(context.Background(), dsn); err != nil {
		t.Fatalf("migrate.Run() err = %v, want nil", err)
	}
}

func levelTestSite(t *testing.T, d *DB) int64 {
	t.Helper()
	ctx := context.Background()
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	siteID, err := d.CreateSite(ctx, NewSite{
		Slug: "lv-" + stamp, Title: "LV", Headline: "LV",
		Domain: "lv-" + stamp + ".test", MediaDomain: "lv-" + stamp + ".test",
		RoleNames: map[string]string{
			"admin": "Admin", "member": "Member",
			"registered": "Registered", "everyone": "Everyone",
		},
	})
	if err != nil {
		t.Fatalf("CreateSite() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := d.pool.Exec(ctx,
			`DELETE FROM magnetar_xp_ledger WHERE user_id IN (SELECT id FROM web_user WHERE username LIKE 'xpuser-%')`); err != nil {
			t.Errorf("clean up ledger err = %v", err)
		}
		for _, sql := range []string{
			`DELETE FROM web_user_roles WHERE role_id IN (SELECT id FROM web_role WHERE site_id = $1)`,
			`DELETE FROM web_role_permissions WHERE role_id IN (SELECT id FROM web_role WHERE site_id = $1)`,
			`DELETE FROM web_role WHERE site_id = $1`,
			`DELETE FROM magnetar_level_config WHERE site_id = $1`,
			`DELETE FROM web_settings WHERE site_id = $1`,
			`DELETE FROM web_site WHERE id = $1`,
		} {
			if _, err := d.pool.Exec(ctx, sql, siteID); err != nil {
				t.Errorf("clean up site %d err = %v", siteID, err)
			}
		}
	})
	// The level config seed runs inside CreateSite; confirm it landed.
	var n int
	if err := d.pool.QueryRow(ctx, `SELECT count(*) FROM magnetar_level_config WHERE site_id = $1`, siteID).Scan(&n); err != nil {
		t.Fatalf("count level config err = %v, want nil", err)
	}
	if n != MaxLevel {
		t.Fatalf("level config rows = %d, want %d", n, MaxLevel)
	}
	return siteID
}

func userRoles(t *testing.T, d *DB, siteID, userID int64) []string {
	t.Helper()
	rows, err := d.pool.Query(context.Background(),
		`SELECT r.slug FROM web_user_roles ur JOIN web_role r ON r.id = ur.role_id
		 WHERE ur.user_id = $1 AND r.site_id = $2 ORDER BY r.slug`, userID, siteID)
	if err != nil {
		t.Fatalf("user roles err = %v, want nil", err)
	}
	defer rows.Close()
	var slugs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan role slug err = %v", err)
		}
		slugs = append(slugs, s)
	}
	return slugs
}

func TestAwardXPFlow(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	migrateLevelTestDB(t, dsn)
	d := newTestDB(t)
	ctx := context.Background()
	siteID := levelTestSite(t, d)

	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	userID, err := d.CreateUser(ctx, "xpuser-"+stamp, "XP User", "hash", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		d.pool.Exec(context.Background(), `DELETE FROM web_user WHERE id = $1`, userID)
	})

	// 1. First award: 50 XP, still level 1.
	leveledUp, level, err := d.AwardXP(ctx, siteID, userID, 50, XPSourceArticleCreate, "article:1")
	if err != nil {
		t.Fatalf("AwardXP() err = %v, want nil", err)
	}
	if leveledUp || level != 1 {
		t.Errorf("AwardXP() = (%v, %d), want (false, 1)", leveledUp, level)
	}

	// 2. Same ref again: deduped, XP unchanged.
	if _, _, err := d.AwardXP(ctx, siteID, userID, 50, XPSourceArticleCreate, "article:1"); err != nil {
		t.Fatalf("AwardXP() duplicate err = %v, want nil", err)
	}
	var xp int64
	if err := d.pool.QueryRow(ctx, `SELECT xp FROM web_user WHERE id = $1`, userID).Scan(&xp); err != nil {
		t.Fatalf("read xp err = %v, want nil", err)
	}
	if xp != 50 {
		t.Errorf("xp after duplicate award = %d, want 50", xp)
	}

	// 3. Another 60 XP (total 110): crosses 100 -> level 2.
	leveledUp, level, err = d.AwardXP(ctx, siteID, userID, 60, XPSourceForumPost, "post:1")
	if err != nil {
		t.Fatalf("AwardXP() err = %v, want nil", err)
	}
	if !leveledUp || level != 2 {
		t.Errorf("AwardXP() = (%v, %d), want (true, 2)", leveledUp, level)
	}
	var dbLevel int
	if err := d.pool.QueryRow(ctx, `SELECT xp, level FROM web_user WHERE id = $1`, userID).Scan(&xp, &dbLevel); err != nil {
		t.Fatalf("read xp/level err = %v, want nil", err)
	}
	if xp != 110 || dbLevel != 2 {
		t.Errorf("web_user = (xp %d, level %d), want (110, 2)", xp, dbLevel)
	}

	// 4. The level-2 role was auto-created and granted.
	roles := userRoles(t, d, siteID, userID)
	found := false
	for _, s := range roles {
		if s == "level-2" {
			found = true
		}
	}
	if !found {
		t.Errorf("user roles = %q, want level-2 granted", roles)
	}
}

func TestSaveLevelConfig(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	migrateLevelTestDB(t, dsn)
	d := newTestDB(t)
	ctx := context.Background()
	siteID := levelTestSite(t, d)

	// Custom prefix + tighter thresholds.
	thresholds := DefaultLevelThresholds
	thresholds[1] = 50 // level 2 at 50 XP
	if err := d.SaveLevelConfig(ctx, siteID, thresholds, "段"); err != nil {
		t.Fatalf("SaveLevelConfig() err = %v, want nil", err)
	}
	if prefix, err := d.LevelPrefix(ctx, siteID); err != nil || prefix != "段" {
		t.Errorf("LevelPrefix() = %q, %v; want %q, nil", prefix, err, "段")
	}
	if got := FormatLevel("段", 3); got != "段3" {
		t.Errorf("FormatLevel = %q, want 段3", got)
	}
	got, err := d.LevelThresholds(ctx, siteID)
	if err != nil {
		t.Fatalf("LevelThresholds() err = %v, want nil", err)
	}
	if LevelForXP(60, got) != 2 {
		t.Errorf("LevelForXP(60) with custom thresholds = %d, want 2", LevelForXP(60, got))
	}

	// Reject non-increasing thresholds and empty prefix.
	bad := DefaultLevelThresholds
	bad[2] = bad[1]
	if err := d.SaveLevelConfig(ctx, siteID, bad, "Lv"); err == nil {
		t.Errorf("SaveLevelConfig() with flat thresholds err = nil, want error")
	}
	if err := d.SaveLevelConfig(ctx, siteID, DefaultLevelThresholds, ""); err == nil {
		t.Errorf("SaveLevelConfig() with empty prefix err = nil, want error")
	}
}
