CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    login TEXT UNIQUE NOT NULL,
    pass_hash TEXT NOT NULL,
    is_admin BOOLEAN
);