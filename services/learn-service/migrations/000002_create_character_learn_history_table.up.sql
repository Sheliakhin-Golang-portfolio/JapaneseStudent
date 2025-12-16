CREATE TABLE IF NOT EXISTS character_learn_history (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    character_id INT NOT NULL,
    hiragana_reading_result FLOAT DEFAULT 0,
    hiragana_writing_result FLOAT DEFAULT 0,
    hiragana_listening_result FLOAT DEFAULT 0,
    katakana_reading_result FLOAT DEFAULT 0,
    katakana_writing_result FLOAT DEFAULT 0,
    katakana_listening_result FLOAT DEFAULT 0,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    UNIQUE KEY unique_user_character (user_id, character_id),
    INDEX idx_user_id (user_id),
    INDEX idx_character_id (character_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

