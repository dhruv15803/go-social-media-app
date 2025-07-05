CREATE TABLE
    IF NOT EXISTS post_images (
        id SERIAL PRIMARY KEY,
        post_image_url TEXT NOT NULL,
        post_id INTEGER NOT NULL,
        FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE
    );