CREATE TABLE
    IF NOT EXISTS likes (
        liked_by_id INTEGER NOT NULL,
        liked_post_id INTEGER NOT NULL,
        liked_at TIMESTAMP DEFAULT NOW (),
        FOREIGN KEY (liked_by_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (liked_post_id) REFERENCES posts (id) ON DELETE CASCADE,
        UNIQUE (liked_by_id, liked_post_id)
    );