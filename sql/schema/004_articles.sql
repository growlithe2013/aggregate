-- +goose Up
CREATE TABLE articles(
                      id UUID PRIMARY KEY,
                      created_at TIMESTAMP NOT NULL,
                      updated_at TIMESTAMP NOT NULL,
                      name TEXT NOT NULL,
                      description TEXT NOT NULL,
                      feed_url TEXT NOT NULL UNIQUE,
                      feed_id UUID NOT NULL,
                      publish_date TIMESTAMP NOT NULL,
                      last_fetched_at TIMESTAMP NOT NULL,
                      FOREIGN KEY (feed_id) REFERENCES feeds(id)
                      ON DELETE CASCADE
);

-- +goose Down
DROP TABLE articles;