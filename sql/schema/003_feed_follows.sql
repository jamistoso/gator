-- +goose Up
CREATE TABLE feed_follows (
    id          UUID        PRIMARY KEY,
    created_at  TIMESTAMP   NOT NULL,
    updated_at  TIMESTAMP   NOT NULL,
    user_id     UUID        REFERENCES  users  
                            ON DELETE CASCADE,
    feed_id     UUID        REFERENCES  feeds
                            ON DELETE CASCADE,
    FOREIGN KEY(user_id)
    REFERENCES users(id),
    FOREIGN KEY(feed_id)
    REFERENCES feeds(id),
    UNIQUE(user_id, feed_id)
);

-- +goose Down
DROP TABLE feeds;