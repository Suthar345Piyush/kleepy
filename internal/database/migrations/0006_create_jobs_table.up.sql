-- table for background jobs - worker queue 

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;


CREATE TABLE IF NOT EXISTS jobs (
   id TEXT PRIMARY KEY,
   job_type TEXT NOT NULL,
   payload TEXT NOT NULL DEFAULT '{}',   -- json {upload_process, transcribe, clip, caption, burn_captions}

   status TEXT NOT NULL DEFAULT 'queued',
   priority TEXT NOT NULL DEFAULT 0,
   attempts TEXT NOT NULL DEFAULT 0,
   max_attempts TEXT NOT NULL DEFAULT 3,
   last_error TEXT,
   queued_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
   started_at DATETIME,
   ended_at DATETIME
);


CREATE INDEX IF NOT EXISTS idx_job_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_job_priority ON jobs(priority DESC, queued_at ASC);

