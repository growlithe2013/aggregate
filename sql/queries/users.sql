-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, name)
VALUES(
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE name = $1;

-- name: GetUserID :one
SELECT id FROM users WHERE name = $1;

-- name: ClearDB :exec
DELETE FROM users;

-- name: GetUsers :many
SELECT name FROM users;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;
