CREATE TABLE IF NOT EXISTS urls (
                                    id SERIAL PRIMARY KEY,
                                    url TEXT NOT NULL UNIQUE,
                                    check_interval_seconds INT NOT NULL DEFAULT 300,
                                    created_at TIMESTAMPTZ DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS checks (
                                      id SERIAL PRIMARY KEY,
                                      url_id INT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    checked_at TIMESTAMPTZ DEFAULT NOW(),
    status_code INT,
    response_time_ms INT,
    error_message TEXT
    );

CREATE TABLE IF NOT EXISTS tags (
                                    id SERIAL PRIMARY KEY,
                                    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS url_tags (
                                        url_id INT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    tag_id INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (url_id, tag_id)
    );

-- Indexes from Day 1
CREATE INDEX IF NOT EXISTS idx_checks_url_id_checked_at ON checks(url_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_checks_checked_at ON checks(checked_at);
CREATE INDEX IF NOT EXISTS idx_url_tags_tag_id ON url_tags(tag_id);