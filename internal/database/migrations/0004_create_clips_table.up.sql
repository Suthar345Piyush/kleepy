-- clips table  - clipped output files  

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS clips (
   id TEXT PRIMARY KEY,
   video_id TEXT NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
   clip_path TEXT NOT NULL,
   start_time TEXT NOT NULL,
   end_time TEXT NOT NULL,
   duration REAL GENERATED ALWAYS AS (end_time - start_time) STORED,
   label TEXT NOT NULL DEFAULT '',
   status TEXT NOT NULL DEFAULT 'pending',
   error_msg TEXT,
   created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
   updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);


-- index 

CREATE INDEX IF NOT EXISTS idx_clips_video ON clips(video_id);


