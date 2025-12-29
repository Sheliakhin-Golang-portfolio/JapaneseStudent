CREATE TABLE IF NOT EXISTS courses (
    id INT PRIMARY KEY AUTO_INCREMENT,
    slug VARCHAR(255) NOT NULL,
    author_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    short_summary TEXT NOT NULL,
    complexity_level ENUM('Absolute beginner', 'Beginner', 'Intermediate', 'Upper Intermediate', 'Advanced') NOT NULL,
    UNIQUE KEY unique_slug (slug),
    INDEX idx_author_id (author_id),
    INDEX idx_complexity_level (complexity_level),
    INDEX idx_title (title)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

