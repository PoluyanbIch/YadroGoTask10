CREATE TABLE comics (
    id INTEGER PRIMARY KEY,
    url TEXT NOT NULL,
    title JSONB NOT NULL DEFAULT '{}',
    alt JSONB NOT NULL DEFAULT '{}',
    description JSONB NOT NULL DEFAULT '{}'
);