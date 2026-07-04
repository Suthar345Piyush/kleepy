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
