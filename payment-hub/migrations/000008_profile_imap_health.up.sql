ALTER TABLE payment_profiles
    ADD COLUMN imap_last_ok_at DATETIME(3) NULL AFTER is_active,
    ADD COLUMN imap_last_error VARCHAR(500) NULL AFTER imap_last_ok_at,
    ADD COLUMN imap_last_checked_at DATETIME(3) NULL AFTER imap_last_error;
