CREATE TABLE
    IF NOT EXISTS bookmarks (
        bookmarked_by_id INTEGER NOT NULL,
        bookmarked_post_id INTEGER NOT NULL,
        bookmarked_at TIMESTAMP DEFAULT NOW (),
        FOREIGN KEY (bookmarked_by_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (bookmarked_post_id) REFERENCES posts (id) ON DELETE CASCADE,
        UNIQUE (bookmarked_by_id, bookmarked_post_id)
    );