-- videos table schema 
-- using WAL - write ahead logging, used for concurrent read and writes, in this readers don't block writers from writing and writers don't block readers from reading

-- REAL - 64 bit floating point number


PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;


CREATE TABLE IF NOT EXISTS videos (
  id TEXT PRIMARY KEY,       -- uuid  
  filename TEXT NOT NULL,
  original_path TEXT NOT NULL,
  duration REAL NOT NULL DEFAULT 0,
  filesize INTEGER NOT NULL DEFAULT 0,
  width INTEGER NOT NULL DEFAULT 0, 
  height INTEGER NOT NULL DEFAULT 0,
  fps REAL NOT NULL DEFAULT 0,
  codec TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending',
  error_msg TEXT,
  created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);



