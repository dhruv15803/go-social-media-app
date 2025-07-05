CREATE TYPE NOTIFICATION_TYPE AS ENUM ('like', 'comment');

CREATE TABLE
    IF NOT EXISTS notifications (
        id SERIAL PRIMARY KEY,
        user_id INTEGER NOT NULL,
        notification_type NOTIFICATION_TYPE NOT NULL,
        actor_id INTEGER NOT NULL,
        notification_created_at TIMESTAMP DEFAULT NOW (),
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE CASCADE
    );