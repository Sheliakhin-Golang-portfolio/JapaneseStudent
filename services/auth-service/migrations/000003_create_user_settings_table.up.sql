CREATE TABLE IF NOT EXISTS user_settings (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT UNIQUE NOT NULL,
    new_word_count INT DEFAULT 20,
    old_word_count INT DEFAULT 20,
    alphabet_learn_count INT DEFAULT 10,
    language VARCHAR(2) DEFAULT 'en',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    CHECK (new_word_count >= 10 AND new_word_count <= 40),
    CHECK (old_word_count >= 10 AND old_word_count <= 40),
    CHECK (alphabet_learn_count >= 5 AND alphabet_learn_count <= 15),
    CHECK (language IN ('en', 'ru', 'de'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

