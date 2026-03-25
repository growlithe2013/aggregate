-- name: InsertArticle :exec
INSERT INTO articles (id, created_at, updated_at, name, feed_url, feed_id, publish_date, last_fetched_at, description)
VALUES(
          $1,
          $2,
          $3,
          $4,
          $5,
          $6,
          $7,
          $8,
          $9
      );

-- name: ClearArticles :exec
DELETE FROM articles;

-- name: GetArticlesByFeedId :many
SELECT * FROM articles WHERE feed_id = $1;