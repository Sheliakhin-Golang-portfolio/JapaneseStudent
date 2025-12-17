CREATE TABLE IF NOT EXISTS words (
    id INT PRIMARY KEY AUTO_INCREMENT,
    word NVARCHAR(40) NOT NULL,
    phonetic_clues VARCHAR(60) NOT NULL,
    russian_translation VARCHAR(60) NOT NULL,
    english_translation VARCHAR(60) NOT NULL,
    german_translation VARCHAR(60) NOT NULL,
    example VARCHAR(255) NOT NULL,
    example_russian_translation VARCHAR(255) NOT NULL,
    example_english_translation VARCHAR(255) NOT NULL,
    example_german_translation VARCHAR(255) NOT NULL,
    easy_period INT NOT NULL,
    normal_period INT NOT NULL,
    hard_period INT NOT NULL,
    extra_hard_period INT NOT NULL,
    INDEX idx_word (word)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

