CREATE TABLE IF NOT EXISTS comics (
    id    INTEGER PRIMARY KEY,
    url   TEXT    NOT NULL,
    words TEXT[]  NOT NULL
);
