CREATE TABLE
    IF NOT EXISTS follow_requests (
        request_sender_id INTEGER NOT NULL,
        request_receiver_id INTEGER NOT NULL,
        request_at TIMESTAMP DEFAULT NOW (),
        FOREIGN KEY (request_sender_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (request_receiver_id) REFERENCES users (id) ON DELETE CASCADE,
        UNIQUE (request_sender_id, request_receiver_id)
    );