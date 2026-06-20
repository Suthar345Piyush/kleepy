// tests for video metadata service

package metadata

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/config"
	"github.com/Suthar345Piyush/videoclippingpipeline/internal/database"
)

// the schema test for video metadata, on videos table

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
	 )
`

// function for setup test db

func setupTestDB(t *testing.T) *sql.DB {

	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")

	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	return db

}

// test for video

// lavfi is audio and video filtering library in ffmpeg

func generateTestVideo(t *testing.T, dir string, withAudio bool) string {
	t.Helper()

	path := filepath.Join(dir, "sample.mp4")

	args := []string{
		"-y", "-f", "lavfi", "-i", "testsrc=duration=2:size=640x480:rate=24",
	}

	if withAudio {
		args = append(args, "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-c:v", "libx264", "aac", "-c:a", "-shortest", path)
	} else {
		args = append(args, "-c:v", "libx264", path)
	}

	cmd := exec.Command("ffmpeg", args...)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate the test video: %v\n%s", err, out)
	}

	return path

}

// tests for actual service

func TestService_ExtractAndStore_Success(t *testing.T) {

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available on the host")
	}

	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available on the host")
	}

	tmpDir := t.TempDir()

	videoPath := generateTestVideo(t, tmpDir, true)

	db := setupTestDB(t)
	queries := database.New(db)

	ctx := context.Background()

	created, err := queries.CreateVideo(ctx, database.CreateVideoParams{
		ID:           "vid-1",
		Filename:     "sample.mp4",
		OriginalPath: videoPath,
		Filesize:     12345,
	})

	if err != nil {
		t.Fatalf("create video failed: %v", err)
	}

	if created.Status != "pending" {
		t.Fatalf("expected initial status 'pending', got %q", created.Status)
	}

	probe := NewProbe(&config.FFmpegConfig{FFprobePath: "ffprobe"})
	srvc := NewService(probe, queries)

	updated, err := srvc.ExtractAndStore(ctx, "vid-1")
	if err != nil {
		t.Fatalf("ExtractAndStore failed: %v", err)
	}

	if updated.Status != "ready" {
		t.Errorf("expected status 'ready', got %q", updated.Status)
	}

	if updated.Codec != "h264" {
		t.Errorf("expected codec 'h264', got %q", updated.Codec)
	}

	if updated.Height != 480 || updated.Width != 640 {
		t.Errorf("got %dx%d, want 640x480", updated.Height, updated.Width)
	}

	if updated.Fps < 23.9 || updated.Fps > 24.1 {
		t.Errorf("got fps %.4f, want somewhere 24.0", updated.Fps)
	}

	if updated.Duration < 1.9 || updated.Duration > 2.1 {
		t.Errorf("got duration %.4f, want 2.0", updated.Duration)
	}

	// getting video after update failed

	fetched, err := queries.GetVideo(ctx, "vid-1")
	if err != nil {
		t.Fatalf("get video after update failed: %v", err)
	}

	if fetched.Width != 640 || fetched.Status != "ready" {
		t.Errorf("row mismatch: width=%d status=%q", fetched.Width, fetched.Status)
	}

}

// tests if the video file is missing

func TestService_ExtractAndStore_MissingFile(t *testing.T) {

	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available on this host")
	}

	db := setupTestDB(t)
	queries := database.New(db)
	ctx := context.Background()

	_, err := queries.CreateVideo(ctx, database.CreateVideoParams{
		ID:           "vid-missing",
		Filename:     "ghost.mp4",
		OriginalPath: "/path/does/not/exist.mp4",
		Filesize:     0,
	})

	if err != nil {
		t.Fatalf("CreateVideo failed: %v", err)
	}

	probe := NewProbe(&config.FFmpegConfig{FFprobePath: "ffprobe"})
	svc := NewService(probe, queries)

	_, err = svc.ExtractAndStore(ctx, "vid-missing")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	fetched, ferr := queries.GetVideo(ctx, "vid-missing")
	if ferr != nil {
		t.Fatalf("GetVideo failed: %v", ferr)
	}
	if fetched.Status != "error" {
		t.Errorf("expected status 'error' after failed extraction, got %q", fetched.Status)
	}
	if !fetched.ErrorMsg.Valid || fetched.ErrorMsg.String == "" {
		t.Errorf("expected non-empty error_msg to be recorded, got %+v", fetched.ErrorMsg)
	}

}

// test if the video has no audio with it

func TestService_ExtractAndStore_NoAudio(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available on this host")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available on this host")
	}

	tmpDir := t.TempDir()
	videoPath := generateTestVideo(t, tmpDir, false)

	db := setupTestDB(t)
	queries := database.New(db)
	ctx := context.Background()

	_, err := queries.CreateVideo(ctx, database.CreateVideoParams{
		ID:           "vid-silent",
		Filename:     "silent.mp4",
		OriginalPath: videoPath,
		Filesize:     999,
	})
	if err != nil {
		t.Fatalf("CreateVideo failed: %v", err)
	}

	probe := NewProbe(&config.FFmpegConfig{FFprobePath: "ffprobe"})
	svc := NewService(probe, queries)

	updated, err := svc.ExtractAndStore(ctx, "vid-silent")
	if err != nil {
		t.Fatalf("ExtractAndStore failed for no-audio file: %v", err)
	}
	if updated.Status != "ready" {
		t.Errorf("expected status 'ready', got %q", updated.Status)
	}

	// removing that video file

	_ = os.Remove(videoPath)

}
