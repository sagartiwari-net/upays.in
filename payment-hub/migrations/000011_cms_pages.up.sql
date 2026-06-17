-- CMS pages for marketing site (Phase 4)

CREATE TABLE IF NOT EXISTS cms_pages (
    id              CHAR(36) PRIMARY KEY,
    slug            VARCHAR(100) NOT NULL UNIQUE,
    title           VARCHAR(200) NOT NULL,
    meta_description VARCHAR(500) NULL,
    body_html       MEDIUMTEXT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft',
    show_in_nav     TINYINT(1) NOT NULL DEFAULT 0,
    nav_label       VARCHAR(100) NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_cms_status (status),
    KEY idx_cms_nav (show_in_nav, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
