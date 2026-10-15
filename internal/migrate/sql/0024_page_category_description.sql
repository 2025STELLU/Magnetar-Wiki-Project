-- compat: compatible

-- Admin-facing description for page categories, editable from the console.
ALTER TABLE web_category ADD COLUMN description text NOT NULL DEFAULT '';
