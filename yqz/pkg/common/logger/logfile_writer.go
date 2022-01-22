//

package logger

import (
	"fmt"
	"os"
	"strings"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/log/rollwriter"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	pluginName            = "logfile"
	pluginType            = "log"
	defaultFieldsSplitter = "|"
	defaultPattern        = "%k=%v"
	defaultCallerSkip     = 8   // More skip steps caused by encapsulation
	defaultMaxSize        = 100 // unit is MB
	defaultMaxBackups     = 10  // the count of reserved log
	skipFIELD             = "skip"
)

func init() {
	log.RegisterWriter(pluginName, &LogFilePlugin{})
}

// Config is the logfile plugin configuration.
type Config struct {
	// LogPath is the path of log file, eg. /yottadb/log/
	LogPath string `yaml:"log_path"`
	// Filename is the filename of log, eg. monitorService.normal.log
	Filename string `yaml:"filename"`
	// RollType is the file rolling type.
	// size splits files by size and it's the default type.
	// time splits files by time.
	RollType string `yaml:"roll_type"`
	// MaxAge is the maximum retention time, the unit is day.
	MaxAge int `yaml:"max_age"`
	// MaxBackups is the maximum number of log files.
	MaxBackups int `yaml:"max_backups"`
	// Compress is whether the log file is compressed.
	Compress bool `yaml:"compress"`

	// The following parameters are only valid when rolling by size.
	// MaxSize is the maximum log file size (in MB).
	MaxSize int `yaml:"max_size"`

	// The following parameters are only valid when rolling by time.
	// TimeUnit is for rolling files by time and supports year/month/day/hour/minute, the default is day.
	TimeUnit log.TimeUnit `yaml:"time_unit"`

	// The following parameters are for logfileEncoder.
	// GoIDKey determines whether to encoder goroutine id in log. When it is nil, it will not encode.
	GoIDKey string `yaml:"goroutineID_key"`
	// Pattern is the pattern to encode key-value pairs in log.
	Pattern string `yaml:"pattern"`
	// FieldsSplitter is the splitter between the pairs in log.
	FieldsSplitter string `yaml:"fields_splitter"`
}

// LogFilePlugin logfile log trpc plugin.
type LogFilePlugin struct {
}

// Type logfile log trpc plugin type.
func (p *LogFilePlugin) Type() string {
	return pluginType
}

// Setup setups a logfile plugin.
func (p *LogFilePlugin) Setup(name string, configDec plugin.Decoder) error {
	// parse config
	if configDec == nil {
		return fmt.Errorf("logfile writer decoder empty")
	}

	decoder, ok := configDec.(*log.Decoder)
	if !ok {
		return fmt.Errorf("logfile writer log decoder type invalid")
	}

	conf := &log.OutputConfig{}
	err := decoder.Decode(&conf)
	if err != nil {
		return err
	}

	cfg := &Config{}
	err = conf.RemoteConfig.Decode(cfg)
	if err != nil {
		return err
	}

	// set default value if parameter is unset
	fixLogfileConfig(cfg)

	if !isValidPattern(cfg.Pattern) {
		return fmt.Errorf("invalid pattern configuration in logfile")
	}

	logfileConfig := conf.FormatConfig
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        GetLogEncoderKey("T", logfileConfig.TimeKey),
		LevelKey:       GetLogEncoderKey("L", logfileConfig.LevelKey),
		NameKey:        GetLogEncoderKey("N", logfileConfig.NameKey),
		CallerKey:      GetLogEncoderKey("C", logfileConfig.CallerKey),
		MessageKey:     GetLogEncoderKey("M", logfileConfig.MessageKey),
		StacktraceKey:  GetLogEncoderKey("S", logfileConfig.StacktraceKey),
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     log.NewTimeEncoder(conf.FormatConfig.TimeFmt),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := NewLogFileEncoder(encoderConfig, conf.CallerSkip+defaultCallerSkip, cfg.Pattern, cfg.FieldsSplitter,
		GetLogEncoderKey("G", cfg.GoIDKey))

	c := zapcore.NewCore(
		encoder,
		zapcore.AddSync(newFileWriterSyncer(cfg)),
		zap.NewAtomicLevelAt(log.Levels[conf.Level]),
	)

	decoder.Core = c
	return nil
}

// GetLogEncoderKey gets user set log field. When empty string get, it will set default key. Set "skip" to skip field.
func GetLogEncoderKey(defKey, key string) string {
	if key == "" {
		return defKey
	} else if key == skipFIELD {
		return ""
	}

	return key
}

// FixLogfileConfig sets default value, if parameter is unset.
func fixLogfileConfig(conf *Config) {
	if conf.FieldsSplitter == "" {
		conf.FieldsSplitter = defaultFieldsSplitter
	}
	if conf.Pattern == "" {
		conf.Pattern = defaultPattern
	}
	if conf.MaxSize == 0 {
		conf.MaxSize = defaultMaxSize
	}
	if conf.MaxBackups == 0 {
		conf.MaxBackups = defaultMaxBackups
	}
}

// It's invalid to include more than one of less than one 'key' and 'value' placeholder in patter.
func isValidPattern(pattern string) bool {
	if strings.Count(pattern, KeyFormat) != 1 {
		return false
	}

	if strings.Count(pattern, ValueFormat) != 1 {
		return false
	}

	return true
}

func newFileWriterSyncer(c *Config) zapcore.WriteSyncer {
	var fileWS zapcore.WriteSyncer

	if c.RollType == "" {
		return zapcore.Lock(os.Stdout)

	} else if c.RollType == "size" {
		// rolling by size
		logFileSizeRollWriter, _ := rollwriter.NewRollWriter(
			c.Filename,
			rollwriter.WithMaxAge(c.MaxAge),
			rollwriter.WithMaxBackups(c.MaxBackups),
			rollwriter.WithCompress(c.Compress),
			rollwriter.WithMaxSize(c.MaxSize),
		)
		fileWS = zapcore.AddSync(logFileSizeRollWriter)
	} else {
		// rolling by time
		logFileTimeRollWriter, _ := rollwriter.NewRollWriter(
			c.Filename,
			rollwriter.WithMaxAge(c.MaxAge),
			rollwriter.WithMaxBackups(c.MaxBackups),
			rollwriter.WithMaxSize(c.MaxSize),
			rollwriter.WithCompress(c.Compress),
			rollwriter.WithRotationTime(c.TimeUnit.Format()),
		)
		fileWS = zapcore.AddSync(logFileTimeRollWriter)
	}

	return fileWS
}
