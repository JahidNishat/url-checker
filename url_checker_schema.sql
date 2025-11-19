--- Table 1: urls
CREATE TABLE urls
(
    id         SERIAL PRIMARY KEY,
    url        TEXT NOT NULL UNIQUE,
    domain     TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

--- Table 2: checks
CREATE TABLE checks
(
    id          SERIAL PRIMARY KEY,
    url_id      INTEGER   NOT NULL REFERENCES urls (id) ON DELETE CASCADE,
    status      INTEGER   NOT NULL,
    duration_ms INTEGER   NOT NULL,
    checked_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    worker_id   TEXT      NOT NULL,
    error_msg   TEXT
);

-- Table 3: tags
CREATE TABLE tags
(
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

--- Table 4: check_tags
CREATE TABLE check_tags
(
    check_id INTEGER NOT NULL REFERENCES checks (id) ON DELETE CASCADE,
    tag_id   INTEGER NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (check_id, tag_id)
);

--- Indexes for performance
CREATE INDEX idx_checks_url_id ON checks (url_id);
CREATE INDEX idx_checks_status ON checks (status);
CREATE INDEX idx_checks_checked_at ON checks (checked_at);
CREATE INDEX idx_checks_status_checked_at ON checks (status, checked_at);
CREATE INDEX idx_urls_domain ON urls (domain);
CREATE INDEX idx_tags_name ON tags (name);
CREATE INDEX idx_check_tags_check_id ON check_tags (check_id);
CREATE INDEX idx_check_tags_tag_id ON check_tags (tag_id);