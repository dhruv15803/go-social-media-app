CREATE TABLE
    IF NOT EXISTS user_invitations (
        token TEXT PRIMARY KEY,
        user_id INTEGER UNIQUE NOT NULL,
        expiration TIMESTAMP NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
    );