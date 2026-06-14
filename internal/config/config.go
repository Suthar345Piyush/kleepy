package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// main parent config

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Storage  StorageConfig  `yaml:"storage"`
	FFmpeg   FFmpegConfig   `yaml:"ffmpeg"`
	Worker   WorkerConfig   `yaml:"worker"`
	Logger   LoggerConfig   `yaml:"logger"`
	Whisper  WhisperConfig  `yaml:"whisper"`
}

// server config

type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// db config

type DatabaseConfig struct {
	Path           string `yaml:"path"`
	Max_Open_Conns int    `yaml:"max_open_conns"`
	Max_Idle_Conns int    `yaml:"max_idle_conns"`
}

// storage cfg

type StorageConfig struct {
	CaptionsDir    string `yaml:"captions_dir"`
	ClipsDir       string `yaml:"clips_dir"`
	TempDir        string `yaml:"temp_dir"`
	TranscriptsDir string `yaml:"transcripts_dir"`
	UploadsDir     string `yaml:"uploads_dir"`
}

// ffmpeg cfg

type FFmpegConfig struct {
	BinaryPath  string `yaml:"binary_path"`
	FFprobePath string `yaml:"ffprobe_path"`
}

// worker cfg

type WorkerConfig struct {
	PoolSize   int           `yaml:"pool_size"`
	QueueSize  int           `yaml:"queue_size"`
	JobTimeout time.Duration `yaml:"job_timeout"`
}

// logger cfg

type LoggerConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	OutputPath string `yaml:"output_path"`
}

// whisper cfg

type WhisperConfig struct {
	ScriptPath  string `yaml:"script_path"`
	PythonPath  string `yaml:"python_path"`
	Model       string `yaml:"model"`
	ComputeType string `yaml:"compute_type"`
	Device      string `yaml:"device"`
	Language    string `yaml:"language"`
}

// load function

func Load(path string) (*Config, error) {

	// reading the path

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: failed to read file: %w", err)
	}

	cfg := &Config{}

	// parsing the data(path)

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: failed to parsed yaml: %w", err)
	}

	cfg.overrideEnvs()
	cfg.setDefualtVals()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: validation failed: %w", err)
	}

	return cfg, nil

}

// validation function

func (c *Config) validate() error {
	if c.Database.Path == "" {
		return fmt.Errorf("database path must not be empty")
	}

	if c.Storage.UploadsDir == "" {
		return fmt.Errorf("uploads must not be empty")
	}

	if c.Storage.ClipsDir == "" {
		return fmt.Errorf("storage clips must not be empty")
	}

	if c.Storage.CaptionsDir == "" {
		return fmt.Errorf("storage captions must not be empty")
	}

	if c.Storage.TranscriptsDir == "" {
		return fmt.Errorf("storage transcription must not be empty")
	}

	if c.Storage.TempDir == "" {
		return fmt.Errorf("storage temp must not be empty")
	}

	if c.Worker.PoolSize <= 0 {
		return fmt.Errorf("worker pool size must be greater than 0")
	}

	if c.Worker.QueueSize <= 0 {
		return fmt.Errorf("queue size must be greater than 0")
	}

	// verifying format and logging level

	switch c.Logger.Level {
	case "debug", "warn", "error", "info":
	default:
		return fmt.Errorf("logger level must be one from debug|warn|error|info, got %q", c.Logger.Level)
	}

	// logger format must be JSON or simple text

	switch c.Logger.Format {
	case "json", "text":
	default:
		return fmt.Errorf("logger format must be one from json|text, got %q", c.Logger.Format)
	}

	return nil

}

// for deployments overriding selected fields by env

func (c *Config) overrideEnvs() {

	if v := os.Getenv("PIPELINE_DB_PATH"); v != "" {
		c.Database.Path = v
	}

	if v := os.Getenv("PIPELINE_SERVER_PORT"); v != "" {
		var port int

		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			c.Server.Port = port
		}

	}

	if v := os.Getenv("PIPELINE_LOG_LEVEL"); v != "" {
		c.Logger.Level = v
	}

	if v := os.Getenv("PIPELINE_WHISPER_MODEL"); v != "" {
		c.Whisper.Model = v
	}

}

// some default values, if any values remain empty

func (c *Config) setDefualtVals() {

	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}

	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}

	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}

	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}

	if c.Database.Max_Open_Conns == 0 {
		c.Database.Max_Open_Conns = 10
	}

	if c.Database.Max_Idle_Conns == 0 {
		c.Database.Max_Idle_Conns = 5
	}

	if c.FFmpeg.BinaryPath == "" {
		c.FFmpeg.BinaryPath = "ffmpeg"
	}

	if c.FFmpeg.FFprobePath == "" {
		c.FFmpeg.FFprobePath = "ffprobe"
	}

	if c.Whisper.PythonPath == "" {
		c.Whisper.PythonPath = "python"
	}

	if c.Whisper.Model == "" {
		c.Whisper.Model = "small"
	}

	if c.Whisper.Device == "" {
		c.Whisper.Device = "cpu"
	}

	if c.Whisper.ComputeType == "" {
		c.Whisper.Device = "int8"
	}

	if c.Worker.PoolSize == 0 {
		c.Worker.PoolSize = 2
	}

	if c.Worker.QueueSize == 0 {
		c.Worker.QueueSize = 30
	}

	if c.Worker.JobTimeout == 0 {
		c.Worker.JobTimeout = 20 * time.Minute
	}

	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}

	if c.Logger.Format == "" {
		c.Logger.Format = "json"
	}

	if c.Logger.OutputPath == "" {
		c.Logger.OutputPath = "stdout"
	}

}

// function to directories are created as expected
// all are storage directories

func (c *Config) directoriesCreation() error {
	dirs := []string{
		c.Storage.CaptionsDir,
		c.Storage.ClipsDir,
		c.Storage.TempDir,
		c.Storage.TranscriptsDir,
		c.Storage.UploadsDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("config: failed to create directory %q: %w", dir, err)
		}
	}

	return nil

}
