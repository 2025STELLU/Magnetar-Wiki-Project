-- compat: compatible

-- Daily check-in state. One row per user; the ledger (source "checkin")
-- stays the anti-farming dedup, this table only tracks the streak.
CREATE TABLE magnetar_checkins (
    user_id          bigint NOT NULL PRIMARY KEY REFERENCES web_user (id) ON DELETE CASCADE,
    last_checkin_date date NOT NULL,
    streak           integer NOT NULL DEFAULT 1,
    total_days       integer NOT NULL DEFAULT 1,
    updated_at       timestamptz NOT NULL DEFAULT now()
);

-- Admin-assigned display styling for usernames. Empty values mean
-- "no styling". A row that ends up fully empty is deleted instead.
CREATE TABLE magnetar_user_name_styles (
    user_id            bigint NOT NULL PRIMARY KEY REFERENCES web_user (id) ON DELETE CASCADE,
    font_family        text NOT NULL DEFAULT '',
    color              text NOT NULL DEFAULT '',
    gradient_from      text NOT NULL DEFAULT '',
    gradient_to        text NOT NULL DEFAULT '',
    gradient_direction text NOT NULL DEFAULT '',
    updated_by         bigint REFERENCES web_user (id) ON DELETE SET NULL,
    updated_at         timestamptz NOT NULL DEFAULT now()
);
