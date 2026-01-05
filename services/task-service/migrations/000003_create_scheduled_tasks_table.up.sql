CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NULL,
    template_id INT NULL,
    url VARCHAR(500) NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    next_run DATETIME NOT NULL,
    previous_run DATETIME NULL,
    active BOOLEAN DEFAULT TRUE,
    cron VARCHAR(100) NOT NULL,
    FOREIGN KEY (template_id) REFERENCES email_templates(id) ON DELETE SET NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_template_id (template_id),
    INDEX idx_active (active),
    INDEX idx_next_run (next_run)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
