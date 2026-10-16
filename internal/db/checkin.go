package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Daily check-in. One check-in per calendar day (site timezone when the
// caller passes it, server local time otherwise — the daily login XP still
// uses server local) pays XPCheckin; every 7th consecutive day pays a
// streak bonus on top. magnetar_checkins tracks the streak; the XP ledger
// stays the anti-farming dedup, so a retried check-in can never double-pay.

const (
	XPCheckin            = 2
	XPCheckinStreakBonus = 10
)

const (
	XPSourceCheckin       = "checkin"
	XPSourceCheckinStreak = "checkin_streak"
)

// CheckinStatus is what the check-in widget shows.
type CheckinStatus struct {
	CheckedIn bool
	Streak    int
	TotalDays int
}

type checkinRow struct {
	last       time.Time
	streak     int
	total      int
	found      bool
	checkedIn  bool
}

var qCheckinGet = register("CheckinGet", `
SELECT last_checkin_date, streak, total_days
FROM magnetar_checkins WHERE user_id = $1`)

var qCheckinUpsert = register("CheckinUpsert", `
INSERT INTO magnetar_checkins (user_id, last_checkin_date, streak, total_days, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (user_id) DO UPDATE SET
    last_checkin_date = EXCLUDED.last_checkin_date,
    streak = EXCLUDED.streak,
    total_days = EXCLUDED.total_days,
    updated_at = now()`)

func (d *DB) getCheckinRow(ctx context.Context, userID int64, loc ...*time.Location) (checkinRow, error) {
	today := time.Now().In(checkinLoc(loc)).Format("2006-01-02")
	var row checkinRow
	err := d.pool.QueryRow(ctx, qCheckinGet, userID).Scan(&row.last, &row.streak, &row.total)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return checkinRow{}, nil
		}
		return checkinRow{}, fmt.Errorf("checkin row: %w", err)
	}
	row.found = true
	row.checkedIn = row.last.Format("2006-01-02") == today
	return row, nil
}

// checkinLoc resolves which "today" a check-in uses: the site timezone the
// webapi passes in, falling back to server local time when absent (keeps
// the db tests and any direct callers working unchanged).
func checkinLoc(loc []*time.Location) *time.Location {
	if len(loc) > 0 && loc[0] != nil {
		return loc[0]
	}
	return time.Local
}

// CheckinStatus reads today's check-in state for a user.
func (d *DB) CheckinStatus(ctx context.Context, userID int64, loc ...*time.Location) (CheckinStatus, error) {
	row, err := d.getCheckinRow(ctx, userID, loc...)
	if err != nil {
		return CheckinStatus{}, err
	}
	return CheckinStatus{CheckedIn: row.checkedIn, Streak: row.streak, TotalDays: row.total}, nil
}

// DoCheckin records today's check-in and pays the XP. Checking in twice on
// the same day returns the current status with xpAwarded 0 and no error.
// Bots and system accounts are refused, mirroring AwardXP.
func (d *DB) DoCheckin(ctx context.Context, userID int64, loc ...*time.Location) (st CheckinStatus, xpAwarded int, err error) {
	var userType string
	if err := d.pool.QueryRow(ctx, `SELECT type FROM web_user WHERE id = $1`, userID).Scan(&userType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CheckinStatus{}, 0, ErrNotFound
		}
		return CheckinStatus{}, 0, fmt.Errorf("do checkin: lookup user %d: %w", userID, err)
	}
	if userType != UserTypeNormal && userType != UserTypeWikidot {
		return CheckinStatus{}, 0, fmt.Errorf("do checkin: user type %q cannot check in", userType)
	}

	row, err := d.getCheckinRow(ctx, userID, loc...)
	if err != nil {
		return CheckinStatus{}, 0, err
	}
	if row.checkedIn {
		return CheckinStatus{CheckedIn: true, Streak: row.streak, TotalDays: row.total}, 0, nil
	}

	now := time.Now().In(checkinLoc(loc))
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	newStreak, newTotal := 1, 1
	if row.found {
		newTotal = row.total + 1
		if row.last.Format("2006-01-02") == yesterday {
			newStreak = row.streak + 1
		}
	}

	// The streak row goes through first in a transaction; the XP itself is
	// paid through AwardXP, whose ledger dedup ("day:"+today) makes a
	// retried check-in unable to double-pay even if this call dies between
	// the two.
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return CheckinStatus{}, 0, fmt.Errorf("do checkin: begin: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, qCheckinUpsert, userID, today, newStreak, newTotal); err != nil {
		return CheckinStatus{}, 0, fmt.Errorf("do checkin: upsert: %w", err)
	}
	// No site context here, same as the daily login XP: the level role is
	// reconciled on the next site-scoped award.
	if _, _, err := d.AwardXP(ctx, 0, userID, XPCheckin, XPSourceCheckin, "day:"+today); err != nil {
		return CheckinStatus{}, 0, fmt.Errorf("do checkin: award xp: %w", err)
	}
	xpAwarded += XPCheckin
	if newStreak%7 == 0 {
		if _, _, err := d.AwardXP(ctx, 0, userID, XPCheckinStreakBonus, XPSourceCheckinStreak, "streak:"+today); err != nil {
			return CheckinStatus{}, 0, fmt.Errorf("do checkin: award streak bonus: %w", err)
		}
		xpAwarded += XPCheckinStreakBonus
	}
	if err := tx.Commit(ctx); err != nil {
		return CheckinStatus{}, 0, fmt.Errorf("do checkin: commit: %w", err)
	}
	return CheckinStatus{CheckedIn: true, Streak: newStreak, TotalDays: newTotal}, xpAwarded, nil
}
