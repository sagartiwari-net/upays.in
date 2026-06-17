ALTER TABLE payment_profiles
    DROP COLUMN imap_last_ok_at,
    DROP COLUMN imap_last_error,
    DROP COLUMN imap_last_checked_at;
