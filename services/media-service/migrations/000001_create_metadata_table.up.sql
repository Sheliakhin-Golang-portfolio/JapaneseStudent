CREATE TABLE IF NOT EXISTS metadata (
    id VARCHAR(255) PRIMARY KEY,
    content_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    url VARCHAR(500) NOT NULL,
    type VARCHAR(50) NOT NULL,
    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

