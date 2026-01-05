CREATE TABLE IF NOT EXISTS scheduled_task_logs (
    id INT PRIMARY KEY AUTO_INCREMENT,
    task_id INT NOT NULL,
    job_id VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL,
    http_status INT NOT NULL,
    error TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES scheduled_tasks(id) ON DELETE CASCADE,
    INDEX idx_task_id (task_id),
    INDEX idx_job_id (job_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
