package db

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
)

// Admin-assigned username styling. Rendered as inline CSS on the name link
// wherever usernames are printed; validated here so renderers can trust it.

type NameStyle struct {
	FontFamily        string
	Color             string
	GradientFrom      string
	GradientTo        string
	GradientDirection string
}

var (
	hexColorRe = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	fontKeepRe = regexp.MustCompile(`[^\w\s,"'\-]`)
)

// ValidHexColor accepts an empty string or a #rgb / #rrggbb color.
func ValidHexColor(s string) bool {
	return s == "" || hexColorRe.MatchString(s)
}

// ValidGradientDirection accepts an empty string or one of the CSS
// linear-gradient directions the admin UI offers.
func ValidGradientDirection(s string) bool {
	switch s {
	case "", "to right", "to left", "to top", "to bottom",
		"to top right", "to top left", "to bottom right", "to bottom left":
		return true
	}
	return false
}

// SanitizeFontFamily caps the length and strips anything that is not a
// plausible font-name character, so the value is safe to drop into CSS.
func SanitizeFontFamily(s string) string {
	if len(s) > 100 {
		s = s[:100]
	}
	return fontKeepRe.ReplaceAllString(s, "")
}

// ValidateNameStyle checks a style before it is stored.
func ValidateNameStyle(s NameStyle) error {
	for _, c := range []struct {
		name, value string
	}{
		{"color", s.Color},
		{"gradient_from", s.GradientFrom},
		{"gradient_to", s.GradientTo},
	} {
		if !ValidHexColor(c.value) {
			return fmt.Errorf("invalid %s %q: want #rgb or #rrggbb", c.name, c.value)
		}
	}
	if !ValidGradientDirection(s.GradientDirection) {
		return fmt.Errorf("invalid gradient_direction %q", s.GradientDirection)
	}
	if (s.GradientFrom == "") != (s.GradientTo == "") {
		return fmt.Errorf("gradient_from and gradient_to must be set together")
	}
	return nil
}

// IsEmpty reports whether the style carries no styling at all.
func (s NameStyle) IsEmpty() bool {
	return s.FontFamily == "" && s.Color == "" &&
		s.GradientFrom == "" && s.GradientTo == "" && s.GradientDirection == ""
}

var qNameStyleGet = register("NameStyleGet", `
SELECT font_family, color, gradient_from, gradient_to, gradient_direction
FROM magnetar_user_name_styles WHERE user_id = $1`)

var qNameStyleUpsert = register("NameStyleUpsert", `
INSERT INTO magnetar_user_name_styles
    (user_id, font_family, color, gradient_from, gradient_to, gradient_direction, updated_by, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (user_id) DO UPDATE SET
    font_family = EXCLUDED.font_family,
    color = EXCLUDED.color,
    gradient_from = EXCLUDED.gradient_from,
    gradient_to = EXCLUDED.gradient_to,
    gradient_direction = EXCLUDED.gradient_direction,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()`)

var qNameStyleDelete = register("NameStyleDelete", `
DELETE FROM magnetar_user_name_styles WHERE user_id = $1`)

// GetNameStyle returns the user's name style, or nil when none is set.
func (d *DB) GetNameStyle(ctx context.Context, userID int64) (*NameStyle, error) {
	var s NameStyle
	err := d.pool.QueryRow(ctx, qNameStyleGet, userID).Scan(
		&s.FontFamily, &s.Color, &s.GradientFrom, &s.GradientTo, &s.GradientDirection)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get name style: %w", err)
	}
	if s.IsEmpty() {
		return nil, nil
	}
	return &s, nil
}

// SetNameStyle stores an admin-assigned style. A fully empty style deletes
// the row instead, so "no styling" and "never styled" look the same.
func (d *DB) SetNameStyle(ctx context.Context, userID int64, style NameStyle, updatedBy int64) error {
	style.FontFamily = SanitizeFontFamily(style.FontFamily)
	if err := ValidateNameStyle(style); err != nil {
		return err
	}
	if style.IsEmpty() {
		if _, err := d.pool.Exec(ctx, qNameStyleDelete, userID); err != nil {
			return fmt.Errorf("delete name style: %w", err)
		}
		return nil
	}
	var by *int64
	if updatedBy != 0 {
		by = &updatedBy
	}
	if _, err := d.pool.Exec(ctx, qNameStyleUpsert, userID,
		style.FontFamily, style.Color, style.GradientFrom, style.GradientTo,
		style.GradientDirection, by); err != nil {
		return fmt.Errorf("set name style: %w", err)
	}
	return nil
}

// FillNameStyles loads name styles for a batch of users in one query.
func (d *DB) FillNameStyles(ctx context.Context, users []*User) error {
	ids := make([]int64, 0, len(users))
	byID := make(map[int64]*User, len(users))
	for _, u := range users {
		if u == nil || u.NameStyle != nil {
			continue
		}
		ids = append(ids, u.ID)
		byID[u.ID] = u
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := d.pool.Query(ctx, `
SELECT user_id, font_family, color, gradient_from, gradient_to, gradient_direction
FROM magnetar_user_name_styles WHERE user_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("fill name styles: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var s NameStyle
		if err := rows.Scan(&id, &s.FontFamily, &s.Color, &s.GradientFrom, &s.GradientTo, &s.GradientDirection); err != nil {
			return fmt.Errorf("fill name styles: scan: %w", err)
		}
		if s.IsEmpty() {
			continue
		}
		if u, ok := byID[id]; ok {
			cp := s
			u.NameStyle = &cp
		}
	}
	return rows.Err()
}
