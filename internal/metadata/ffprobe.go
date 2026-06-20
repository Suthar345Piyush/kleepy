/*

using ffprobe to analyze the video and audio data from video file, extracting metadata from video like - framerate, bitrate, resolution (width, height), video duration, codec (coder-encoder), codec type - video or audio

not using any third party library like go-ffprobe, using os/exec package to manually getting metadata by the unmarshalling the video metadata struct into JSON format

have to run ffprobe on video filepath to extract metadata

*/

package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/config"
)

// probe struct will run ffprobe binary on config and parses its json output into video metadata struct

type Probe struct {
	ffprobePath string
	timeout     time.Duration
}

// newprobe function to build a probe from ffmpegconfig

func NewProbe(cfg *config.FFmpegConfig) *Probe {
	path := cfg.FFprobePath

	if path == "" {
		path = "ffprobe"
	}

	return &Probe{
		ffprobePath: path,
		timeout:     30 * time.Second,
	}
}

// with timeout return the probe with a fixed timeout

func (p *Probe) WithTimeout(d time.Duration) *Probe {
	if d <= 0 {
		d = 30 * time.Second
	}

	return &Probe{
		ffprobePath: p.ffprobePath,
		timeout:     d,
	}
}

// extract function will run ffprobe command on the filepath to extract the metadata and returned the parsed metadata

func (p *Probe) Extract(ctx context.Context, filePath string) (*VideoMetadata, error) {

	// check if file path is empty or not

	if filePath == "" {
		return nil, fmt.Errorf("ffprobe: file path must not be empty")
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)

	defer cancel()

	// ffprobe command arguments

	args := []string{
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}

	cmd := exec.CommandContext(ctx, p.ffprobePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ffprobe: timed out after %s probing %q", p.timeout, filePath)
		}

		// error returned due to command failed
		return nil, fmt.Errorf("ffprobe: command failed for %q: %w (stderr: %s)", filePath, err, strings.TrimSpace(stderr.String()))

	}

	// output

	var out ffprobeOutput

	// parsing json output

	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("ffprobe: failed to parse json output for %q: %w", filePath, err)
	}

	meta, err := parseProbeOutput(out)

	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w (file %q)", err, filePath)
	}

	return meta, nil

}

// seperate function to parse the probe output and converting it into video metadata struct

func parseProbeOutput(out ffprobeOutput) (*VideoMetadata, error) {

	meta := &VideoMetadata{}

	// video stream and audio stream
	var videoStream *ffprobeStream
	var audioStream *ffprobeStream

	// iterate on the output streams for codec types

	for i := range out.Streams {
		s := &out.Streams[i]

		switch s.CodecType {
		case "video":
			if videoStream == nil {
				videoStream = s
			}

		case "audio":
			if audioStream == nil {
				audioStream = s
			}
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found")
	}

	// codec name and it's dimension (width and height)

	meta.Codec = videoStream.CodecName
	meta.Height = videoStream.Height
	meta.Width = videoStream.Width

	// FPS

	fps, err := parseFrameRate(videoStream)

	if err != nil {
		return nil, fmt.Errorf("failed to parse frame rate: %w", err)
	}

	meta.FPS = fps

	// video duration, taking video's own duration, otherwise take from out

	durStr := videoStream.Duration

	if durStr == "" {
		durStr = out.Format.Duration
	}

	dur, err := parseFloat(durStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}

	meta.Duration = dur

	// bitrate, taking video's own bitrate, otherwise take from out

	btrStr := videoStream.BitRate

	if btrStr == "" {
		btrStr = out.Format.BitRate
	}

	if btr, err := strconv.ParseInt(btrStr, 10, 64); err != nil {
		meta.BitRate = btr
	}

	if audioStream != nil {
		meta.HasAudio = true
		meta.AudioCodec = videoStream.CodecName
	}

	return meta, nil

}

/*

frame rate parse function

FPS - framerate format = num / deno, returns an float

two types of framerate are their 1. avg_frame_rate (average frame-rate) , 2. r_frame_rate (real/base frame-rate)
we prefer the avg one and can fallback to real as well

*/

func parseFrameRate(s *ffprobeStream) (float64, error) {
	raw := s.AvgFrameRate

	if raw == "" || raw == "0/0" {
		raw = s.RFrameRate
	}

	if raw == "" {
		return 0, fmt.Errorf("no framerate field present")
	}

	parts := strings.SplitN(raw, "/", 2)

	if len(parts) != 2 {
		return parseFloat(raw)
	}

	num, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid framerate neumerator %q: %w", parts[0], err)
	}

	deno, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid framerate denominator %q: %w", parts[1], err)
	}

	if deno == 0 {
		return 0, fmt.Errorf("frame rate denominator is zero")
	}

	return num / deno, nil

}

// parse float helper function

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)

	if s == "" {
		return 0, nil
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float value %q: %w", s, err)
	}
	return v, nil
}
