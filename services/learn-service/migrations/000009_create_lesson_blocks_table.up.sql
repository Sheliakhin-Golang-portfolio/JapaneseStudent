CREATE TABLE IF NOT EXISTS lesson_blocks (
    id INT PRIMARY KEY AUTO_INCREMENT,
    lesson_id INT NOT NULL,
    block_type ENUM('video', 'audio', 'text', 'document', 'list') NOT NULL,
    block_order INT NOT NULL,
    block_data JSON NOT NULL,
    FOREIGN KEY (lesson_id) REFERENCES lessons(id) ON DELETE CASCADE,
    INDEX idx_lesson_id (lesson_id),
    INDEX idx_block_order (lesson_id, block_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

