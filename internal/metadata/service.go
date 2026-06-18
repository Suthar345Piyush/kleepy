// service file will join the probe and the database query

// we just extract the data and store it into the database

package metadata

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/database"
)

type Service struct {
	probe   *Probe
	queries *database.Queries
}

// new service function will return a service with probe and sqlc query

func NewService(probe *Probe, queries *database.Queries) *Service {
	return &Service{probe: probe, queries: queries}
}

// extract and store function to run ffprobe on video original path, and then write the video metadata result into the video row using our queries

func (s *Service) ExtractAndStore(ctx context.Context, videoID string) (database.Video, error) {

	video, err := s.queries.GetVideo(ctx, videoID)
	if err != nil {
		return database.Video{}, fmt.Errorf("metadata: failed to load video %q: %w", videoID, err)
	}

	// getting the metadata from extract function from ffprobe file

	meta, err := s.probe.Extract(ctx, video.OriginalPath)
	if err != nil {

		// row doesn't stuck on 'pending' status, on failure case, will fallback to 'error' status
		// updateVideoStatus

		if _, updateErr := s.queries.UpdateVideoStatus(ctx, database.UpdateVideoStatusParams{
			Status:   "error",
			ErrorMsg: sql.NullString{String: err.Error(), Valid: true},
			ID:       videoID,
		}); updateErr != nil {

			return database.Video{}, fmt.Errorf("metadata: ffprobe failed (%v) and failed to record error status: %w", err, updateErr)
		}

		return database.Video{}, fmt.Errorf("metadata: ffprobe extraction failed for video %q: %w", videoID, err)
	}

	updated, err := s.queries.UpdateVideoMetadata(ctx, database.UpdateVideoMetadataParams{
		Duration: meta.Duration,
		Width:    int64(meta.Width),
		Height:   int64(meta.Height),
		Fps:      meta.FPS,
		Codec:    meta.Codec,
		ID:       videoID,
	})

	if err != nil {
		return database.Video{}, fmt.Errorf("metadata: failed to store metadata for video %q: %w", videoID, err)
	}

	return updated, nil

}
