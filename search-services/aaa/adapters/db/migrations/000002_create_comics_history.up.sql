CREATE TABLE comics_history (
    id SERIAL PRIMARY KEY,
    comic_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL
);