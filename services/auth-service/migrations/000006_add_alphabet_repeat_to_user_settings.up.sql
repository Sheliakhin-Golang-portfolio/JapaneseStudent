ALTER TABLE user_settings
ADD COLUMN alphabet_repeat VARCHAR(20) DEFAULT 'in question',
ADD CONSTRAINT chk_alphabet_repeat CHECK (alphabet_repeat IN ('in question', 'ignore', 'repeat'));
