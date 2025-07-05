CREATE TABLE
    IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        email TEXT UNIQUE NOT NULL,
        username TEXT UNIQUE NOT NULL,
        image_url TEXT,
        password TEXT NOT NULL,
        bio TEXT,
        location TEXT,
        date_of_birth DATE NOT NULL,
        is_public BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT NOW (),
        updated_at TIMESTAMP
    );