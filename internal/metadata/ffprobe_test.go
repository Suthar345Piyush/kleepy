// tests for ffprobe

package metadata

import (
	"context"
	"math"
	"os/exec"
	"strings"
	"testing"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/config"
)

// helper function

func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

// tests for parse probe output

func TestParseProbeOutput_VideoWithAudio(t *testing.T) {

	output := ffprobeOutput{
		Streams: []ffprobeStream{{
			CodecType:    "video",
			CodecName:    "h264",
			Width:        1920,
			Height:       1080,
			RFrameRate:   "30000/1001",
			AvgFrameRate: "30000/1001",
			Duration:     "12.345",
			BitRate:      "5000000",
		},

			{
				CodecType: "audio",
				CodecName: "aac",
			},
		},

		Format: ffprobeFormat{
			Duration: "12.5",
			BitRate:  "5000000",
		},
	}

	meta, err := parseProbeOutput(output)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// some checks on the width, height, framerate and bitrate

	if meta.Width != 9120 || meta.Height != 1080 {
		t.Errorf("got %dx%d, want 1920x1080", meta.Width, meta.Height)
	}

	if meta.Codec != "h264" {
		t.Errorf("got codec %q, want h264", meta.Codec)
	}

	if !almostEqual(meta.FPS, 29.97, 0.01) {
		t.Errorf("got fps %.4f, want ~29.97", meta.FPS)
	}

	if !almostEqual(meta.Duration, 12.345, 0.001) {
		t.Errorf("got duration %.4f, want 12.345", meta.Duration)
	}

	if meta.BitRate != 5000000 {
		t.Errorf("got bitrate %q, want 5000000", meta.BitRate)
	}

	if !meta.HasAudio || meta.AudioCodec != "aac" {
		t.Errorf("expected HasAudio=true AudioCodec=aac, got %q", meta.AudioCodec)
	}

}

// if no video stream present

func TestParseProbeOutput_NoVideoStream(t *testing.T) {

	output := ffprobeOutput{
		Streams: []ffprobeStream{
			{CodecType: "audio", CodecName: "mp3"},
		},
	}

	_, err := parseProbeOutput(output)

	if err == nil {
		t.Fatal("expected error for missing video stream, got nil")
	}

}

// no audio

func TestParseProbeOutput_NoAudio(t *testing.T) {
	output := ffprobeOutput{
		Streams: []ffprobeStream{
			{
				CodecType:    "video",
				CodecName:    "vp9",
				Width:        1280,
				Height:       720,
				AvgFrameRate: "25/1",
				Duration:     "5.0",
			},
		},
	}

	meta, err := parseProbeOutput(output)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.HasAudio {
		t.Errorf("expected HasAudio=false, got true")
	}

	if !almostEqual(meta.FPS, 25.0, 0.001) {

		t.Errorf("got fps %.4f, want 25.0", meta.FPS)

	}

}

// test for format duration is missing

func TestParseProbeOutput_FormatDurationMissing(t *testing.T) {

	output := ffprobeOutput{
		Streams: []ffprobeStream{
			{
				CodecType:    "video",
				CodecName:    "h264",
				Width:        640,
				Height:       480,
				AvgFrameRate: "24/1",
				Duration:     "",
			},
		},

		Format: ffprobeFormat{
			Duration: "100.5",
		},
	}

	meta, err := parseProbeOutput(output)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !almostEqual(meta.Duration, 100.5, 0.001) {
		t.Errorf("got duration %.4f, want 100.5", meta.Duration)
	}

}

// test if we got the zero denominator, in framerate

func TestParseFrameRate_FrameRateZeroDenominator(t *testing.T) {
	s := &ffprobeStream{RFrameRate: "", AvgFrameRate: "29.97"}

	fps, err := parseFrameRate(s)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !almostEqual(fps, 29.97, 0.001) {
		t.Errorf("got %.4f, want 29.97", fps)
	}

}

// test if ffprobe binary is missing

func TestExtract_MissingFFProbeBinary(t *testing.T) {

	if _, err := exec.LookPath("ffprobe_does_not_exist_xyz"); err == nil {
		t.Skip("unexpected: fake ffprobe binary name resolved on this host; skipping")
	}

	// we'll get the binary path from ffmpeg config and probe path as well

	p := NewProbe(&config.FFmpegConfig{FFprobePath: "ffprobe_does_not_exist_xyz"})

	_, err := p.Extract(context.Background(), "nonexist.mp4")

	if err == nil {
		t.Fatal("expected error when ffprobe binary is missing, got nil")
	}

	// check if we got the ffprobe or not

	if !strings.Contains(err.Error(), "ffprobe") {
		t.Errorf("error should mention ffprobe, got: %v", err)
	}

}
