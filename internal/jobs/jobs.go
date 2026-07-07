// package jobs contains standard job queue system

package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/database"
)

// job type itself in string format, cause in db it is TEXT

type JobType string

/*
we have specific job types for tasks like

- process video job
- clip cut job
- transcribe job
- srt file generation job
- captions job

*/

const (
	JobTypeProcessVideo      JobType = "process_video"
	JobTypeClipCut           JobType = "clip_cut"
	JobTypeTranscribe        JobType = "transcribe"
	JobTypeSRTFileGeneration JobType = "srt_file_generation"
	JobTypeBurnCaptions      JobType = "burn_captions"
)

// process job type video payload

type ProcessVideoPayload struct {
	VideoID string `json:"video_id"`
}

// clip cut job struct

type ClipCutPayload struct {
	ClipID  string `json:"clip_id"`
	VideoID string `json:"video_id"`
}

// transcribe job struct

type TranscribePayload struct {
	VideoID      string `json:"video_id"`
	TranscriptID string `json:"transcript_id"`
}

// srt file generation

type SRTFileGenerationPayload struct {
	TranscriptID string `json:"transcript_id"`
	ClipID       string `json:"clip_id"`
}

// burn captions payload

type BurnCaptionPayload struct {
	ClipID    string `json:"clip_id"`
	CaptionID string `json:"caption_id"`
}

// some payload helper functions

// marshal payload for encoding any payload struct into json format, it will return an error if their is any error

func MarshalPayload(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("jobs: failed to marshal payload: %w", err)
	}

	return string(b), nil
}

// unmarshal payload for decoding the JSON string

func UnMarshalPayload(raw string, v any) error {
	if err := json.Unmarshal([]byte(raw), v); err != nil {
		return fmt.Errorf("jobs: failed to unmarshal payload %q: %w", raw, err)
	}

	return nil

}

// handler interface, will be like used by all jobs pipeline for processing

// if this return nil -> then pool marks the jobs as done
// if this returns error -> it records the error and retry until attempts reached the max_attempts

type Handler interface {
	Handle(ctx context.Context, job database.Job) error
}

// handler function is that which implement the Handler interface

type HandlerFunc func(ctx context.Context, job database.Job) error

func (f HandlerFunc) Handle(ctx context.Context, job database.Job) error {
	return f(ctx, job)
}

// registry will map the job type with their handlers

// key - JobType, value - Handler
type Registry struct {
	handlers map[JobType]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[JobType]Handler)}
}

// register function will connects the job types with their handlers
// it will panic if the same job trying to get registered twice

func (r *Registry) Register(jobType JobType, h Handler) {

	// if that job type exists twice
	if _, exists := r.handlers[jobType]; exists {
		panic(fmt.Sprintf("jobs: handler already registered for job type %q", jobType))
	}

	r.handlers[jobType] = h

}

// get function will return the handler for jobtype
// ok boolean for conforming that we got the handler for that job type or not

func (r *Registry) Get(jobType JobType) (Handler, bool) {
	h, ok := r.handlers[jobType]

	return h, ok
}

// getting all the job types which are registered
// will return all the jobs in contained in slice

func (r *Registry) Registered() []JobType {
	types := make([]JobType, 0, len(r.handlers))

	for t := range r.handlers {
		types = append(types, t)
	}

	return types

}
