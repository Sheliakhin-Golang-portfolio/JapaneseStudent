CREATE TABLE IF NOT EXISTS dictionary_history (
    id INT PRIMARY KEY AUTO_INCREMENT,
    word_id INT NOT NULL,
    user_id INT NOT NULL,
    next_appearance DATE NOT NULL,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE,
    UNIQUE KEY unique_user_word (user_id, word_id),
    INDEX idx_user_id (user_id),
    INDEX idx_word_id (word_id),
    INDEX idx_next_appearance (next_appearance)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

