// tests for timeline service

// tests on segments and

package timeline

import (
	"context"
	"database/sql"
	"math"
	"testing"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/database"
)

// test schema

const testSchema = `
CREATE TABLE videos (
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

CREATE TABLE clips (
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
);`

// function for setting up DB

func setupTestDB(t *testing.T) (*sql.DB, *database.Queries) {

	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")

	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	return db, database.New(db)

}

// ready video function, both creating video and updating video metadata

func readyFunc(t *testing.T, id string, duration float64, q *database.Queries) {
	t.Helper()

	ctx := context.Background()

	// creating the video first

	_, err := q.CreateVideo(ctx, database.CreateVideoParams{
		ID:           id,
		Filename:     "test.mp4",
		OriginalPath: "/tmp/test.mp4",
		Filesize:     1000,
	})

	if err != nil {
		t.Fatalf("error in creating video: %v", err)
	}

	// updating the video content (metadata)

	_, err = q.UpdateVideoMetadata(ctx, database.UpdateVideoMetadataParams{
		Duration: duration,
		Width:    1920,
		Height:   1080,
		Fps:      30,
		Codec:    "h264",
		ID:       id,
	})

	if err != nil {
		t.Fatalf("Updated video metadata failed: %v", err)
	}

}

// helper function, to check between float values

func isEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

// testing the video segment on 90 sec video, it will have three parts 0-30, 30-60, 60-90

func TestBuildSegments_FixedParts(t *testing.T) {

	ops := Options{ClipLength: 30, MinLength: 15, Overlap: 0}

	// final video duration will be 90 secs
	segments := buildSegments(90, ops)

	if len(segments) != 3 {
		t.Fatalf("got %d segments, want only 3", len(segments))
	}

	cases := [][2]float64{{0, 30}, {30, 60}, {60, 90}}

	for i, w := range segments {
		if w.Index != i {
			t.Errorf("window %d: Index=%d", i, w.Index)
		}

		if !isEqual(w.StartTime, cases[i][0]) || !isEqual(w.EndTime, cases[i][1]) {
			t.Errorf("window %d: got [%.3f, %.3f], want [%.3f, %.3f]", i, w.StartTime, w.EndTime, cases[i][0], cases[i][1])
		}

		if !isEqual(w.Duration, 30) {
			t.Errorf("segment duration should be 30: %v", w.Duration)
		}

	}

}
