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
	Port          int           `yaml:"port"`
	Host          string        `yaml:"host"`
	Read_Timeout  time.Duration `yaml:"read_timeout"`
	Write_Timeout time.Duration `yaml:"write_timeout"`
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
	PoolSize   int `yaml:"pool_size"`
	QueueSize  int `yaml:"queue_size"`
	JobTimeout int `yaml:"job_timeout"`
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

func (c *Config) setDefualts() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}

}
