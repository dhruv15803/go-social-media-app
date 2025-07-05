CREATE TABLE
    IF NOT EXISTS posts (
        id SERIAL PRIMARY KEY,
        post_content TEXT NOT NULL,
        user_id INTEGER NOT NULL,
        parent_post_id INTEGER,
        post_created_at TIMESTAMP DEFAULT NOW (),
        post_updated_at TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (parent_post_id) REFERENCES posts (id) ON DELETE CASCADE
    );