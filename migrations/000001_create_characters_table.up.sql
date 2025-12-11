CREATE TABLE IF NOT EXISTS characters (
    id INT PRIMARY KEY AUTO_INCREMENT,
    consonant VARCHAR(10) NOT NULL,
    vowel VARCHAR(10) NOT NULL,
    english_reading VARCHAR(50) NOT NULL,
    russian_reading VARCHAR(50) NOT NULL,
    katakana VARCHAR(10) NOT NULL,
    hiragana VARCHAR(10) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_consonant (consonant),
    INDEX idx_vowel (vowel),
    INDEX idx_katakana (katakana),
    INDEX idx_hiragana (hiragana)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;



