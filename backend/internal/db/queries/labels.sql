-- name: ListLabels :many
SELECT * FROM labels WHERE household_id = $1 ORDER BY lower(name);

-- name: CreateLabel :one
INSERT INTO labels (id, household_id, name, color)
VALUES (@id, @household_id, @name, @color)
RETURNING *;

-- name: UpdateLabel :one
UPDATE labels
SET name  = coalesce(sqlc.narg(name), name),
    color = coalesce(sqlc.narg(color), color)
WHERE id = @id AND household_id = @household_id
RETURNING *;

-- name: DeleteLabel :execrows
DELETE FROM labels WHERE id = @id AND household_id = @household_id;

-- name: CountLabelsByIDs :one
SELECT count(*) FROM labels WHERE household_id = @household_id AND id = ANY(@ids::uuid[]);
