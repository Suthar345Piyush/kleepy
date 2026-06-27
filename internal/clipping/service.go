// service will orchestrate the single clip cut, it will take the clip row from db and after cutting clip writes back to the row

package clipping

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/database"
	"github.com/Suthar345Piyush/videoclippingpipeline/internal/logger"
)

// service struct connection with the db

// with cutter and preset for encoding the video

type Service struct {
	queries *database.Queries
	cutter  *Cutter
	preset  EncodePreset
}

// service function returns service struct with default preset set to it

func NewService(queries *database.Queries, cutter *Cutter, preset EncodePreset) *Service {

	if preset.VideoCodec == "" {
		preset = DefaultPreset
	}

	return &Service{
		queries: queries,
		cutter:  cutter,
		preset:  preset,
	}

}

/*

function to process the clip using the clipID , it will work like this :

-> it will load the clip row and verify it's status is pending (waiting)

-> then it will load the parent video row, for the source path

-> then it will parse the start time and end time for clip cutting

-> after checks for output directory exists or not

-> it will call the cutter

-> it clip cut is done successfully then, status = 'done'

-> if not then status = 'error' with an error message

*/

// this function will return the updated database.clip row  at the end

func (s *Service) ProcessClip(ctx context.Context, clipID string) (database.Clip, error) {

	// logger

	log := logger.FromContext(ctx).With(slog.String("clip_id", clipID))

	// loading the clip row from db

	clip, err := s.queries.GetClip(ctx, clipID)

	if err != nil {
		return database.Clip{}, fmt.Errorf("clipping: failed to load the clip: %w", clipID, err)
	}

	// verification for the clip status 'pending', processing those, which have status pending

	if clip.Status != "pending" {
		return clip, fmt.Errorf("clipping: clip %q has status %q, expected 'pending'", clipID, clip.Status)
	}

	// loading the parent video row, for the source path of the video

	video, err := s.queries.GetVideo(ctx, clip.VideoID)

	if err != nil {
		return database.Clip{}, s.markError(ctx, clipID, fmt.Errorf("clipping: failed to load video %q for the clip %q: %w", clip.VideoID, clipID, err))
	}

	// after video loading, extracting the  start and end timestamp of the video clip

	startTime, err := strconv.ParseFloat(clip.StartTime, 64)
	if err != nil {
		return database.Clip{}, s.markError(ctx, clipID, fmt.Errorf("clipping: invalid starttime %q in the clip %q: %w", clip.StartTime, clipID, err))
	}

	endTime, err := strconv.ParseFloat(clip.EndTime, 64)
	if err != nil {
		return database.Clip{}, s.markError(ctx, clipID, fmt.Errorf("clipping: invalid endtime %q in the clip %q: %w", clip.EndTime, clipID, err))
	}

	// checking that output directory exists in disk or not

	outDir := filepath.Dir(clip.ClipPath)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return database.Clip{}, s.markError(ctx, clipID, fmt.Errorf("clipping: failed to create output dir of %q: %w", outDir, err))
	}

	// cut part

	log.Info("cutting clip", slog.String("source", video.OriginalPath), slog.String("output", clip.ClipPath), slog.Float64("start", startTime), slog.Float64("end", endTime))

	ci := ClipInput{
		SourcePath: video.OriginalPath,
		OutputPath: clip.ClipPath,
		StartTime:  startTime,
		EndTime:    endTime,
		Preset:     s.preset,
	}

	if _, err := s.cutter.Cut(ctx, ci); err != nil {
		return database.Clip{}, s.markError(ctx, clipID, err)
	}

	// if clip cut done successfully then status to done

	// we have to update the clip status

	updated, err := s.queries.UpdateClipsStatus(ctx, database.UpdateClipsStatusParams{
		Status:   "done",
		ErrorMsg: sql.NullString{Valid: false},
		ID:       clipID,
	})

	if err != nil {
		return database.Clip{}, fmt.Errorf("clipping: cut succeeded but failed to update the status to done", clipID, err)
	}

	log.Info("clip done", slog.String("path", updated.ClipPath))

	return updated, nil

}

// separate error logger, it will log any row's status 'error', so that process clip function can use it to show the original error , and keep itself out the error logging

func (s *Service) markError(ctx context.Context, clipID string, err error) error {

	log := logger.FromContext(ctx)

	log.Error("clip failed", slog.String("clip_id", clipID), slog.String("error", err.Error()))

	if _, dbErr := s.queries.UpdateClipsStatus(ctx, database.UpdateClipsStatusParams{
		Status:   "error",
		ErrorMsg: sql.NullString{String: err.Error(), Valid: true},
		ID:       clipID,
	}); dbErr != nil {
		log.Error("also failed to record the error status", slog.String("clip_id", clipID), slog.String("db_err", dbErr.Error()))
	}

	return err

}
