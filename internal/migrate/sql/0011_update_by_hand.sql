-- compat: compatible
ALTER TABLE magnetar_update ADD COLUMN scheduled_by_hand boolean NOT NULL DEFAULT false;
