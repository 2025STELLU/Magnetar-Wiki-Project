-- compat: compatible
ALTER TABLE web_user ADD COLUMN xp bigint NOT NULL DEFAULT 0;
ALTER TABLE web_user ADD COLUMN level integer NOT NULL DEFAULT 1;
ALTER TABLE web_settings ADD COLUMN level_prefix text NOT NULL DEFAULT 'Lv';

-- Ledger of XP awards. The primary key is the anti-farming dedup:
-- one (user, source, ref) can only ever pay out once. One-time actions use
-- a stable ref (e.g. the article id); daily actions fold the date into the
-- ref (e.g. "day:2026-10-13").
CREATE TABLE magnetar_xp_ledger (
    user_id    bigint NOT NULL REFERENCES web_user (id) ON DELETE CASCADE,
    day        date NOT NULL,
    source     text NOT NULL,
    ref_id     text NOT NULL DEFAULT '',
    amount     integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, source, ref_id)
);
CREATE INDEX magnetar_xp_ledger_user_idx ON magnetar_xp_ledger (user_id);

-- Per-site level configuration. Levels are 1..10; xp_required is the
-- total XP needed to REACH that level. Admins edit these (and the
-- level_prefix in web_settings) from the console; each level's
-- permissions are the permission set of its "level-N" role.
CREATE TABLE magnetar_level_config (
    site_id     bigint NOT NULL REFERENCES web_site (id) ON DELETE CASCADE,
    level       integer NOT NULL CHECK (level BETWEEN 1 AND 10),
    xp_required bigint NOT NULL,
    PRIMARY KEY (site_id, level)
);

-- Seed defaults for existing sites.
INSERT INTO magnetar_level_config (site_id, level, xp_required)
SELECT s.id, l.level, l.xp_required
FROM web_site s
CROSS JOIN (VALUES
    (1, 0), (2, 100), (3, 300), (4, 600), (5, 1000),
    (6, 1500), (7, 2100), (8, 2800), (9, 3600), (10, 4500)
) AS l(level, xp_required);
