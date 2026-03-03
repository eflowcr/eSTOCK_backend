-- Presentations CRUD for sqlc
-- Schema: db/migrations (presentations table)

-- name: ListPresentations :many
SELECT presentation_id, description
FROM presentations
ORDER BY description ASC;

-- name: GetPresentationByID :one
SELECT presentation_id, description
FROM presentations
WHERE presentation_id = $1
LIMIT 1;

-- name: PresentationExistsByID :one
SELECT EXISTS(SELECT 1 FROM presentations WHERE presentation_id = $1) AS exists;

-- name: CreatePresentation :one
INSERT INTO presentations (presentation_id, description)
VALUES ($1, $2)
RETURNING presentation_id, description;

-- name: UpdatePresentation :one
UPDATE presentations
SET description = $2
WHERE presentation_id = $1
RETURNING presentation_id, description;

-- name: DeletePresentation :exec
DELETE FROM presentations WHERE presentation_id = $1;
