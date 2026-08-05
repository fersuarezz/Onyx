CREATE TABLE diary_entries (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    text TEXT NOT NULL DEFAULT '',
    date DATE NOT NULL
);

CREATE TABLE attachments (
    entry_id INT NOT NULL,
    dir TEXT NOT NULL,
    PRIMARY KEY (entry_id, dir),
    FOREIGN KEY (entry_id) REFERENCES diary_entries (id) ON DELETE CASCADE
);