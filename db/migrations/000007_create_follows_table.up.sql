CREATE TABLE
    IF NOT EXISTS follows (
        follower_id INTEGER NOT NULL,
        following_id INTEGER NOT NULL,
        followed_at TIMESTAMP DEFAULT NOW (),
        FOREIGN KEY (follower_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (following_id) REFERENCES users (id) ON DELETE CASCADE,
        UNIQUE (follower_id, following_id)
    );