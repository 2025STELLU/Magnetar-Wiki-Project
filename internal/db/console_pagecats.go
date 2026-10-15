package db

import (
	"context"
	"fmt"
)

// PageCategory is the slim shape the console page-category manager works
// with. The old admin edits far more per category (indexing, rating modes,
// themes, per-role overrides); the console only needs name and description.
type PageCategory struct {
	ID          int64
	Name        string
	Description string
}

var qConsolePageCategories = register("ConsolePageCategories", `
SELECT id, name, COALESCE(description, '')
FROM web_category
WHERE site_id = $1
ORDER BY name, id`)

// ConsolePageCategories lists the site's page categories for the console.
func (d *DB) ConsolePageCategories(ctx context.Context, siteID int64) ([]PageCategory, error) {
	rows, err := d.pool.Query(ctx, qConsolePageCategories, siteID)
	if err != nil {
		return nil, fmt.Errorf("list page categories: %w", err)
	}
	defer rows.Close()

	out := []PageCategory{}
	for rows.Next() {
		var c PageCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Description); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var qConsolePageCategoryInsert = register("ConsolePageCategoryInsert", `
INSERT INTO web_category (name, description, is_indexed, site_id)
VALUES ($1, $2, true, $3)`)

var qConsolePageCategoryUpdate = register("ConsolePageCategoryUpdate", `
UPDATE web_category SET name = $1, description = $2
WHERE id = $3 AND site_id = $4`)

// ConsoleSavePageCategory creates (id == 0) or renames/describes a category.
// It deliberately leaves indexing, rating modes, themes and permission
// overrides alone; those stay in the old admin. A missing web_settings row
// is fine: readers treat it as "follow the site".
func (d *DB) ConsoleSavePageCategory(ctx context.Context, siteID, id int64, name, description string) error {
	if id == 0 {
		if _, err := d.pool.Exec(ctx, qConsolePageCategoryInsert, name, description, siteID); err != nil {
			return fmt.Errorf("create page category %q: %w", name, err)
		}
		return nil
	}
	if _, err := d.pool.Exec(ctx, qConsolePageCategoryUpdate, name, description, id, siteID); err != nil {
		return fmt.Errorf("update page category %d: %w", id, err)
	}
	return nil
}
