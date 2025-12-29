CREATE TABLE IF NOT EXISTS lessons (
    id INT PRIMARY KEY AUTO_INCREMENT,
    slug VARCHAR(255) NOT NULL,
    course_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    short_summary TEXT NOT NULL,
    `order` INT NOT NULL,
    UNIQUE KEY unique_slug (slug),
    FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    INDEX idx_course_id (course_id),
    INDEX idx_order (course_id, `order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

