// package jobs contains standard job queue system

package jobs

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
