-- compat: compatible

-- The 0002/0003/0008/0010/0011/0018/0021 migrations were briefly rewritten to
-- create magnetar_* tables directly; they are restored to their original
-- pwikit_* form so databases coming from ProjectWikit keep working. This
-- migration performs the rename for those databases. It is a no-op on
-- databases that already have the magnetar_* tables.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pwikit_user_address')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'magnetar_user_address') THEN
        ALTER TABLE pwikit_user_address RENAME TO magnetar_user_address;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pwikit_admin_log')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'magnetar_admin_log') THEN
        ALTER TABLE pwikit_admin_log RENAME TO magnetar_admin_log;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pwikit_member_sanction')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'magnetar_member_sanction') THEN
        ALTER TABLE pwikit_member_sanction RENAME TO magnetar_member_sanction;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pwikit_update')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'magnetar_update') THEN
        ALTER TABLE pwikit_update RENAME TO magnetar_update;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pwikit_claim_code')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'magnetar_claim_code') THEN
        ALTER TABLE pwikit_claim_code RENAME TO magnetar_claim_code;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'pwikit_message_clear')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'magnetar_message_clear') THEN
        ALTER TABLE pwikit_message_clear RENAME TO magnetar_message_clear;
    END IF;
END $$;
