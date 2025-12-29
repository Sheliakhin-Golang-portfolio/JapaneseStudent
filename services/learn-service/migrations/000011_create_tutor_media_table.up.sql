CREATE TABLE IF NOT EXISTS tutor_media (
    id INT PRIMARY KEY AUTO_INCREMENT,
    tutor_id INT NOT NULL,
    slug VARCHAR(255) NOT NULL,
    media_type ENUM('video', 'doc', 'audio') NOT NULL,
    url VARCHAR(500) NOT NULL,
    UNIQUE KEY unique_slug (slug),
    INDEX idx_tutor_id (tutor_id),
    INDEX idx_media_type (media_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

