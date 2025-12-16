CREATE TABLE IF NOT EXISTS characters (
    id INT PRIMARY KEY AUTO_INCREMENT,
    consonant VARCHAR(1) NOT NULL,
    vowel VARCHAR(1) NOT NULL,
    english_reading VARCHAR(3) NOT NULL,
    russian_reading VARCHAR(3) NOT NULL,
    katakana VARCHAR(1) NOT NULL,
    hiragana VARCHAR(1) NOT NULL,
    INDEX idx_consonant (consonant),
    INDEX idx_vowel (vowel),
    INDEX idx_katakana (katakana),
    INDEX idx_hiragana (hiragana)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;