-- captions table - output as srt(subrip) file  

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;



CREATE TABLE IF NOT EXISTS captions (
  id TEXT PRIMARY KEY,
  clip_id TEXT NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
  format TEXT NOT NULL DEFAULT 'srt',
  caption_path TEXT NOT NULL,
  burned_id INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);


CREATE INDEX IF NOT EXISTS idx_captions_clip ON captions(clip_id);