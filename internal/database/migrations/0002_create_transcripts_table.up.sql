-- transcripts table - up 
-- using pragma (journal-mode = wal)
-- allowed foreign keys constraint
-- whisper output will be our transcript



PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS transcripts (
   id TEXT PRIMARY KEY,
   video_id TEXT NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
   language TEXT NOT NULL DEFAULT 'en',
   model TEXT NOT NULL DEFAULT 'small',
   full_text TEXT NOT NULL DEFAULT '',
   transcript_path TEXT,
   status TEXT NOT NULL DEFAULT 'pending',
   error_msg TEXT,
   created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
   updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);



