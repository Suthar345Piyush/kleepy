-- name: CreateVideo :one
INSERT INTO videos (id, filename, original_path, filesize, status) VALUES (?, ?, ?, ?, 'pending') RETURNING *;

-- name: GetVideo :one
SELECT * FROM videos WHERE id = ? LIMIT 1;

-- name: ListVideos :many
SELECT * FROM videos ORDER BY created_at DESC;

-- name: UpdateVideoMetadata :one
UPDATE videos SET duration = ?, width = ?, height = ?, fps = ?, codec = ?, status = 'ready', updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ? RETURNING *;

-- name: UpdateVideoStatus :one
UPDATE videos SET status = ?, error_msg = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ? RETURNING *;

-- name: DeleteVideo :exec
DELETE FROM videos WHERE id = ?;




-- name: CreateTranscript :one
INSERT INTO transcripts (id, video_id, language, model, status) VALUES (?, ?, ?, 'small', 'pending') RETURNING *;


-- name: GetTranscript :one
SELECT * FROM transcripts WHERE id = ? LIMIT 1;


-- name: GetTranscriptByVideo :one
SELECT * FROM transcripts WHERE video_id = ? LIMIT 1;


-- name: UpdateTranscriptDone :one
UPDATE transcripts SET full_text = ?, transcript_path = ?, status = 'done',  updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ? RETURNING *;


-- name: UpdateTranscriptStatus :one
UPDATE transcripts SET status = ?, error_msg = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ? RETURNING *;



-- name: InsertSegment :one
INSERT INTO segments (transcript_id, start_time, end_time, text, confidence) VALUES (?, ?, ?, ?, ?) RETURNING *;


-- name: ListSegmentsByTranscript :many
SELECT * FROM segments WHERE transcript_id = ? ORDER BY start_time ASC;


-- name: DeleteSegmentByTranscript :exec
DELETE FROM segments WHERE transcript_id = ?;



-- name: CreateClip :one
INSERT INTO clips (id, video_id, clip_path, start_time, end_time, label, status) VALUES (?, ?, ?, ?, ?, ?, 'pending') RETURNING *;


-- name: GetClip :one
SELECT * FROM clips WHERE id = ? LIMIT 1;


-- name: ListClipsByVideo :many
SELECT * FROM clips WHERE video_id = ? ORDER BY start_time ASC;


-- name: UpdateClipsStatus :one
UPDATE clips SET status = ?, error_msg = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ? RETURNING *;


-- name: DeleteClip :exec
DELETE FROM clips WHERE id = ?;



-- name: CreateCaption :one
INSERT INTO captions (id, clip_id, format, caption_path, burned_id) VALUES (?, ?, 'srt', ?, ?) RETURNING *;


-- name: GetCaption :one
SELECT * FROM captions WHERE id = ? LIMIT 1;

-- name: ListCaptionsByClip :many
SELECT * FROM captions WHERE clip_id = ? ORDER BY created_at ASC;


-- name: DeleteCaption :exec
DELETE FROM captions WHERE id = ?;



-- name: EnqueueJob :one
INSERT INTO jobs (id, job_type, payload, priority) VALUES (?, ?, ?, ?) RETURNING *;


-- name: GetJob :one
SELECT * FROM jobs WHERE id = ? LIMIT 1;



-- name: DequeueNextJob :one
UPDATE jobs SET status = 'running', started_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'), attempts = attempts + 1 WHERE id = (
   SELECT id FROM jobs WHERE status = 'queued' AND attempts < max_attempts ORDER BY priority DESC, queued_at ASC LIMIT 1
) RETURNING *;



-- name: CompleteJob :one
UPDATE jobs SET status = 'done', ended_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ? RETURNING *;


-- name: FailJob :one
UPDATE jobs SET status = 'failed', last_error = ?, ended_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ? RETURNING *;


-- name: RequeueJob :one
UPDATE jobs SET status = 'queued' WHERE id = ? RETURNING *;


-- name: ListJobsByStatus :many
SELECT * FROM jobs WHERE status = ? ORDER BY queued_at DESC;


-- name: CancelJob :one
UPDATE jobs SET status = 'cancelled', ended_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ? RETURNING *;


