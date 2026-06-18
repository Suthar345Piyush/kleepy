// data structs (models) for metadata package

package metadata

// main video metadata struct, ffprobe output

type VideoMetadata struct {
	Duration   float64
	Width      int
	Height     int
	FPS        float64
	Codec      string // video codec - h264
	AudioCodec string // audio codec - aac
	BitRate    int64  // bits/second
	HasAudio   bool
}

// struct for incoming ffprobe stream

type ffprobeStream struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	RFrameRate   string `json:"r_frame_rate"`
	AvgFrameRate string `json:"avg_frame_rate"`
	Duration     string `json:"duration"`
	BitRate      string `json:"bit_rate"`
}

// struct for ffprobe format

type ffprobeFormat struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
}

/*

ffprobe output contains slice of streams and format
streams like - codec info (h264, aac, mp3...), dimensions, framerate(fps), bitrate, duration, pixel format, sample rate and channels for audio streams
format - filename, format-name, duration in seconds, size in bytes, bitrate, etc....

we want output in JSON  with command  "ffprobe -print_format json"

*/

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}
