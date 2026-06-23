// video timeline service based on clip segments and timeline options

package timeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"path/filepath"
	"strconv"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/database"
)

//  service struct

type Service struct {
	ClipsDir string // from config's -> storageConfig
	queries  *database.Queries
}

// function to build a service

func NewService(clipsDir string, queries *database.Queries) *Service {

	return &Service{ClipsDir: clipsDir, queries: queries}

}

// main service function - generate and store in db with pending status and in one row

// it will return clips

func (s *Service) GenerateAndStore(opts Options, ctx context.Context, videoID string) ([]database.Clip, error) {

	if err := opts.validate(); err != nil {
		return nil, err
	}

	// getting/loading the video

	video, err := s.queries.GetVideo(ctx, videoID)

	if err != nil {
		return nil, fmt.Errorf("timeline: failed to load the video %q: %w", videoID, err)
	}

	// generating video timeline only once, when ffprobe runs successfully, otherwise video will stay on the pending status and don't have correct duration of the video as well

	// if the video status is not "ready", it means video not probed successfully, so return a error, for running metadata extraction first (ffprobe - metadata package(ffprobe.go))

	if video.Status != "ready" {
		return nil, fmt.Errorf("timeline: video %q not probed status is %q, want status to be 'ready' - run metadata extraction first", video.Status, videoID)
	}

	if video.Duration <= 0 {
		return nil, fmt.Errorf("timeline: video %q has duration %.3f, should be greater than 0", videoID, video.Duration)
	}

	// fixed sized segments

	segments := buildSegments(video.Duration, opts)

	// validation on segment

	if len(segments) == 0 {
		return nil, fmt.Errorf("timeline: no clip segment is produced for video %q (duration = %.3fs, clip length = %.3fs, min length = %.3fs)", videoID, video.Duration, opts.ClipLength, opts.MinLength)
	}

	// clips  making, clip length should be equal to segments

	clips := make([]database.Clip, 0, len(segments))

	// range iterating on segments
	// and generating clip id's

	for _, i := range segments {
		id, err := generateID()
		if err != nil {
			return nil, fmt.Errorf("timeline: failed to generate clip id: %w", err)
		}

		// creating clip row for particular segment

		clip, err := s.queries.CreateClip(ctx, database.CreateClipParams{
			ID:        id,
			VideoID:   videoID,
			ClipPath:  clipPathFunc(s.ClipsDir, videoID, i),
			StartTime: formatFloat(i.StartTime),
			EndTime:   formatFloat(i.EndTime),
			Label:     i.Label,
		})

		if err != nil {
			return nil, fmt.Errorf("timeline: failed to create row for segment %d (%.3f-%.3f): %w", i.Index, i.StartTime, i.EndTime, err)
		}

		clips = append(clips, clip)

	}

	return clips, nil

}

// function - formatFloat

func formatFloat(sec float64) string {

	return strconv.FormatFloat(sec, 'f', 3, 64)

}

// clip path function

func clipPathFunc(clipsDir, videoID string, i Segments) string {

	filename := fmt.Sprintf("clip_%d_%05d-%05d.mp4", i.Index, int(math.Round(i.StartTime*1000)), int(math.Round(i.EndTime*1000)))

	return filepath.Join(clipsDir, videoID, filename)
}

// function to generate random id, using stdlib's crypto random

// random id with prefix cl-feegrgergbiufwy37434

func generateID() (string, error) {

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto random not worked: %w", err)
	}

	return "cl-" + hex.EncodeToString(b), nil

}

// function for fixed segments
// it will returns an clipped segments, according to the overlap value

// eg - overlap = 0, clip length = 30sec, the segment will be (0, 30) , (30, 60), (60, 90)...

// if the overlap = 5, clip length = 30sec, then (0, 30) , (25, 55), (50, 80) .....

// xtra = clip length - overlap

func buildSegments(videoDuration float64, ops Options) []Segments {

	xtra := ops.ClipLength - ops.Overlap

	var segments []Segments

	index := 0

	// iterating till video duration
	// every iteration it goes to (xtra + start)

	for start := 0.0; start < videoDuration; start = roundTo3(xtra + start) {

		end := math.Min(roundTo3(start+ops.ClipLength), videoDuration)
		segmentDuration := roundTo3(end - start)

		// if our segment length(duration) is less than min length of clip then break

		if segmentDuration < ops.MinLength {
			break
		}

		segments = append(segments, Segments{
			Index:     index,
			StartTime: start,
			EndTime:   end,
			Label:     labelFunc(index, start, end),
		})
		index++
	}

	return segments

}

// round off function

func roundTo3(f float64) float64 {
	return math.Round(f*1000) / 1000
}

// label function start time, end time,

// returns the format -> clip_0_00:00:00-00:00:30

// start and end should be in the "hours:minutes:seconds" format - HH:MM:SS

func labelFunc(index int, start, end float64) string {

	return fmt.Sprintf("clip_%d_%s-%s", index, timeFormat(start), timeFormat(end))

}

// function for start time and end time format -  HH:MM:SS

func timeFormat(sec float64) string {
	total := int(math.Round(sec))
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
