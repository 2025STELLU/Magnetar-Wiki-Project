package db

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

// Database integration tests for the check-in system and name styles.
// Skipped without MAGNETAR_TEST_DSN; the schema is migrated in-test.

func checkinTestUser(t *testing.T, d *DB, prefix string) int64 {
	t.Helper()
	ctx := context.Background()
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	userID, err := d.CreateUser(ctx, prefix+"-"+stamp, prefix+" User", "hash", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		d.pool.Exec(ctx, `DELETE FROM magnetar_checkins WHERE user_id = $1`, userID)
		d.pool.Exec(ctx, `DELETE FROM magnetar_user_name_styles WHERE user_id = $1`, userID)
		d.pool.Exec(ctx, `DELETE FROM magnetar_xp_ledger WHERE user_id = $1`, userID)
		d.pool.Exec(ctx, `DELETE FROM web_user WHERE id = $1`, userID)
	})
	return userID
}

func userXP(t *testing.T, d *DB, userID int64) int64 {
	t.Helper()
	var xp int64
	if err := d.pool.QueryRow(context.Background(),
		`SELECT xp FROM web_user WHERE id = $1`, userID).Scan(&xp); err != nil {
		t.Fatalf("read xp err = %v", err)
	}
	return xp
}

func backdateCheckin(t *testing.T, d *DB, userID int64, daysAgo, streak, total int) {
	t.Helper()
	date := time.Now().AddDate(0, 0, -daysAgo).Format("2006-01-02")
	if _, err := d.pool.Exec(context.Background(),
		`INSERT INTO magnetar_checkins (user_id, last_checkin_date, streak, total_days)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id) DO UPDATE SET last_checkin_date = $2, streak = $3, total_days = $4`,
		userID, date, streak, total); err != nil {
		t.Fatalf("backdate checkin err = %v", err)
	}
}

func TestDoCheckinFlow(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	migrateLevelTestDB(t, dsn)
	d := newTestDB(t)
	ctx := context.Background()
	userID := checkinTestUser(t, d, "checkin")

	// 1. First check-in: streak 1, total 1, +2 XP.
	st, xp, err := d.DoCheckin(ctx, userID)
	if err != nil {
		t.Fatalf("DoCheckin() err = %v, want nil", err)
	}
	if !st.CheckedIn || st.Streak != 1 || st.TotalDays != 1 {
		t.Errorf("DoCheckin() status = %+v, want {true 1 1}", st)
	}
	if xp != XPCheckin || userXP(t, d, userID) != XPCheckin {
		t.Errorf("DoCheckin() xp = %d (db %d), want %d", xp, userXP(t, d, userID), XPCheckin)
	}

	// 2. Same day again: idempotent, no XP.
	st, xp, err = d.DoCheckin(ctx, userID)
	if err != nil {
		t.Fatalf("DoCheckin() duplicate err = %v, want nil", err)
	}
	if !st.CheckedIn || st.Streak != 1 || xp != 0 {
		t.Errorf("DoCheckin() duplicate = (%+v, %d), want (streak 1, xp 0)", st, xp)
	}
	if got := userXP(t, d, userID); got != XPCheckin {
		t.Errorf("xp after duplicate = %d, want %d", got, XPCheckin)
	}

	// 3. Consecutive day: streak grows.
	backdateCheckin(t, d, userID, 1, 1, 1)
	st, xp, err = d.DoCheckin(ctx, userID)
	if err != nil {
		t.Fatalf("DoCheckin() day 2 err = %v, want nil", err)
	}
	if st.Streak != 2 || st.TotalDays != 2 {
		t.Errorf("DoCheckin() day 2 status = %+v, want streak 2 total 2", st)
	}

	// 4. Broken streak: resets to 1, total keeps counting.
	backdateCheckin(t, d, userID, 2, 5, 5)
	st, _, err = d.DoCheckin(ctx, userID)
	if err != nil {
		t.Fatalf("DoCheckin() broken err = %v, want nil", err)
	}
	if st.Streak != 1 || st.TotalDays != 6 {
		t.Errorf("DoCheckin() broken status = %+v, want streak 1 total 6", st)
	}
}

func TestDoCheckinStreakBonus(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	migrateLevelTestDB(t, dsn)
	d := newTestDB(t)
	ctx := context.Background()
	userID := checkinTestUser(t, d, "streak")

	// Yesterday was day 6 of a streak; today completes day 7.
	backdateCheckin(t, d, userID, 1, 6, 6)
	before := userXP(t, d, userID)
	st, xp, err := d.DoCheckin(ctx, userID)
	if err != nil {
		t.Fatalf("DoCheckin() err = %v, want nil", err)
	}
	if st.Streak != 7 {
		t.Errorf("DoCheckin() streak = %d, want 7", st.Streak)
	}
	if want := XPCheckin + XPCheckinStreakBonus; xp != want {
		t.Errorf("DoCheckin() xp = %d, want %d", xp, want)
	}
	if got := userXP(t, d, userID) - before; got != int64(XPCheckin+XPCheckinStreakBonus) {
		t.Errorf("db xp delta = %d, want %d", got, XPCheckin+XPCheckinStreakBonus)
	}
	var n int
	if err := d.pool.QueryRow(ctx,
		`SELECT count(*) FROM magnetar_xp_ledger WHERE user_id = $1 AND source = $2`,
		userID, XPSourceCheckinStreak).Scan(&n); err != nil {
		t.Fatalf("count streak ledger err = %v", err)
	}
	if n != 1 {
		t.Errorf("streak bonus ledger rows = %d, want 1", n)
	}

	// Day 8: no bonus.
	backdateCheckin(t, d, userID, 1, 7, 7)
	_, xp, err = d.DoCheckin(ctx, userID)
	if err != nil {
		t.Fatalf("DoCheckin() day 8 err = %v, want nil", err)
	}
	if xp != XPCheckin {
		t.Errorf("DoCheckin() day 8 xp = %d, want %d (no bonus)", xp, XPCheckin)
	}
}

func TestDoCheckinRefusesBots(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	migrateLevelTestDB(t, dsn)
	d := newTestDB(t)
	ctx := context.Background()
	userID := checkinTestUser(t, d, "botuser")
	if _, err := d.pool.Exec(ctx,
		`UPDATE web_user SET type = 'bot' WHERE id = $1`, userID); err != nil {
		t.Fatalf("make bot err = %v", err)
	}
	if _, _, err := d.DoCheckin(ctx, userID); err == nil {
		t.Errorf("DoCheckin() bot err = nil, want refusal")
	}
	if _, _, err := d.DoCheckin(ctx, 999999999); err == nil {
		t.Errorf("DoCheckin() missing user err = nil, want ErrNotFound")
	}
}

func TestCheckinStatusFresh(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	migrateLevelTestDB(t, dsn)
	d := newTestDB(t)
	ctx := context.Background()
	userID := checkinTestUser(t, d, "freshcheckin")
	st, err := d.CheckinStatus(ctx, userID)
	if err != nil {
		t.Fatalf("CheckinStatus() err = %v, want nil", err)
	}
	if st.CheckedIn || st.Streak != 0 || st.TotalDays != 0 {
		t.Errorf("CheckinStatus() fresh = %+v, want zeros", st)
	}
}

func TestNameStyleValidation(t *testing.T) {
	for _, s := range []string{"", "#fff", "#FFF", "#a1b2c3", "#A1B2C3"} {
		if !ValidHexColor(s) {
			t.Errorf("ValidHexColor(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"fff", "#ff", "#fffff", "#gggggg", "red", "#12345g"} {
		if ValidHexColor(s) {
			t.Errorf("ValidHexColor(%q) = true, want false", s)
		}
	}
	for _, s := range []string{"", "to right", "to bottom left"} {
		if !ValidGradientDirection(s) {
			t.Errorf("ValidGradientDirection(%q) = false, want true", s)
		}
	}
	if ValidGradientDirection("diagonal") {
		t.Errorf("ValidGradientDirection(diagonal) = true, want false")
	}
	if got := SanitizeFontFamily("Noto Sans, \"Hi\"; color:red"); got != "Noto Sans, \"Hi\" colorred" {
		t.Errorf("SanitizeFontFamily() = %q, want stripped", got)
	}
	if len(SanitizeFontFamily(string(make([]byte, 200)))) > 100 {
		t.Errorf("SanitizeFontFamily() did not truncate to 100")
	}
	// Gradient ends must come in pairs.
	if err := ValidateNameStyle(NameStyle{GradientFrom: "#fff"}); err == nil {
		t.Errorf("ValidateNameStyle() half gradient err = nil, want error")
	}
	if err := ValidateNameStyle(NameStyle{
		GradientFrom: "#fff", GradientTo: "#000",
		GradientDirection: "to right", Color: "#abc",
	}); err != nil {
		t.Errorf("ValidateNameStyle() full gradient err = %v, want nil", err)
	}
	if err := ValidateNameStyle(NameStyle{Color: "nope"}); err == nil {
		t.Errorf("ValidateNameStyle() bad color err = nil, want error")
	}
}

func TestSetGetNameStyle(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	migrateLevelTestDB(t, dsn)
	d := newTestDB(t)
	ctx := context.Background()
	userID := checkinTestUser(t, d, "namestyle")
	adminID := checkinTestUser(t, d, "styleadmin")

	if got, err := d.GetNameStyle(ctx, userID); err != nil || got != nil {
		t.Fatalf("GetNameStyle() fresh = (%v, %v), want (nil, nil)", got, err)
	}
	style := NameStyle{
		FontFamily: "Noto Serif", Color: "#a1b2c3",
		GradientFrom: "#ffffff", GradientTo: "#000000", GradientDirection: "to right",
	}
	if err := d.SetNameStyle(ctx, userID, style, adminID); err != nil {
		t.Fatalf("SetNameStyle() err = %v, want nil", err)
	}
	got, err := d.GetNameStyle(ctx, userID)
	if err != nil {
		t.Fatalf("GetNameStyle() err = %v, want nil", err)
	}
	if got == nil || *got != style {
		t.Errorf("GetNameStyle() = %+v, want %+v", got, style)
	}
	// scanUser fills the style on the user struct.
	u, err := d.UserByID(ctx, userID)
	if err != nil {
		t.Fatalf("UserByID() err = %v, want nil", err)
	}
	if u.NameStyle == nil || *u.NameStyle != style {
		t.Errorf("UserByID().NameStyle = %+v, want %+v", u.NameStyle, style)
	}
	// All-empty clears the row.
	if err := d.SetNameStyle(ctx, userID, NameStyle{}, adminID); err != nil {
		t.Fatalf("SetNameStyle() clear err = %v, want nil", err)
	}
	if got, err := d.GetNameStyle(ctx, userID); err != nil || got != nil {
		t.Errorf("GetNameStyle() after clear = (%v, %v), want (nil, nil)", got, err)
	}
	// Invalid input is rejected before touching the DB.
	if err := d.SetNameStyle(ctx, userID, NameStyle{Color: "bogus"}, adminID); err == nil {
		t.Errorf("SetNameStyle() bad color err = nil, want error")
	}
}

func TestSearchUsers(t *testing.T) {
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", EnvDSN)
	}
	migrateLevelTestDB(t, dsn)
	d := newTestDB(t)
	ctx := context.Background()
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	u1 := checkinTestUser(t, d, "findme-"+stamp)
	u2 := checkinTestUser(t, d, "alsomatch-"+stamp)
	if _, err := d.pool.Exec(ctx,
		`UPDATE web_user SET display_name = $2 WHERE id = $1`, u2, "FindMe-"+stamp+" Display"); err != nil {
		t.Fatalf("set display name err = %v", err)
	}

	results, err := d.SearchUsers(ctx, "findme-"+stamp, 20)
	if err != nil {
		t.Fatalf("SearchUsers() err = %v, want nil", err)
	}
	seen := map[int64]bool{}
	for _, r := range results {
		seen[r.ID] = true
	}
	if !seen[u1] || !seen[u2] {
		t.Errorf("SearchUsers() missed users: got %d rows, want both %d and %d", len(results), u1, u2)
	}
	// LIKE metacharacters in the query are literal.
	if _, err := d.SearchUsers(ctx, "%_%", 20); err != nil {
		t.Errorf("SearchUsers() metachar err = %v, want nil", err)
	}
	// Short garbage matches nothing new and doesn't error.
	results, err = d.SearchUsers(ctx, "zz-no-such-user-"+stamp, 20)
	if err != nil {
		t.Fatalf("SearchUsers() empty err = %v, want nil", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchUsers() empty = %d rows, want 0", len(results))
	}
}
