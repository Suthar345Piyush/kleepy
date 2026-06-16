-- table for segments - timestamps based 

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS segments (
   id INTEGER PRIMARY KEY AUTOINCREMENT,
   transcript_id TEXT NOT NULL REFERENCES transcripts(id) ON DELETE CASCADE,
   start_time REAL NOT NULL, 
   end_time REAL NOT NULL,
   text TEXT NOT NULL,
   confidence REAL NOT NULL DEFAULT 0
);

-- index 

CREATE INDEX IF NOT EXISTS idx_segments_transcript ON segments(transcript_id);

