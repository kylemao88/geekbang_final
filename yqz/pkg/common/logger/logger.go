//

// Package logger is used to record information.
// In this package, we include the following log type:
// - Normal includes log level of Trace, Debug, and Info.                                                  [un-report]
// - Wf, it includes log level of Warn, Error, and Fatal.                                                  [report]
// - SYSTEM records log about the start-stop, configuration loading, internal task execution and so on.    [un-report]
// - API records log every API call. The API type includes Read, Write, and Admin.                         [report]
// - Stat records log the statistics data of the program over a period of time.                            [report]
//
// [report] represents the log data will be reported to third-party platform.
// [un-report] represents the opposite.
package logger

import (
	"os"
	"strconv"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
)

func init() {
	// Register
	plugin.Register("sys", log.DefaultLogFactory)
	plugin.Register("wf", log.DefaultLogFactory)
	plugin.Register("api", log.DefaultLogFactory)
	plugin.Register("stat", log.DefaultLogFactory)

	// init default loggers without configuration
	callerSkip := 2
	normalLogger = log.NewZapLogWithCallerSkip(defaultConfig[normalLog], callerSkip)
	systemLogger = log.NewZapLogWithCallerSkip(defaultConfig[systemLog], callerSkip)
	wfLogger = log.NewZapLogWithCallerSkip(defaultConfig[wfLog], callerSkip)
	apiLogger = log.NewZapLogWithCallerSkip(defaultConfig[apiLog], callerSkip)
	statLogger = log.NewZapLogWithCallerSkip(defaultConfig[statLog], callerSkip)
	log.Register(normalLog, normalLogger)
	log.Register(systemLog, systemLogger)
	log.Register(wfLog, wfLogger)
	log.Register(apiLog, apiLogger)
	log.Register(statLog, statLogger)

	// log.trace level is always enabled.
	log.EnableTrace()

	// Log level disabled or not by environment variable
	traceEnabled = DisableFromEnv(logTraceEnv)
	debugEnabled = DisableFromEnv(logDebugEnv)
	infoEnabled = DisableFromEnv(logInfoEnv)
	apiWriteEnabled = DisableFromEnv(logAPIWriteEnv)
	apiReadEnabled = DisableFromEnv(logAPIReadEnv)
}

// default loggers' related parameters
var (
	normalLogger log.Logger
	systemLogger log.Logger
	wfLogger     log.Logger
	apiLogger    log.Logger
	statLogger   log.Logger
)
var defaultLogConf = []log.OutputConfig{
	{
		Writer: "logfile",
		Level:  "debug",
	},
}
var defaultConfig = map[string][]log.OutputConfig{
	normalLog: defaultLogConf,
	systemLog: defaultLogConf,
	wfLog:     defaultLogConf,
	apiLog:    defaultLogConf,
	statLog:   defaultLogConf,
}

// logger type name
const (
	normalLog = "default" // logs log to xxx.normal.log file, it includes level Trace, Debug, and Info.
	systemLog = "sys"     // logs log to xxx.sys.log as Error level.
	wfLog     = "wf"      // logs log to xxx.wf.log, it includes level Warn, Error, and Fatal.
	// apiLog logs log to xxx.api.log, it includes type Read(Debug level), Write(Info level) and Admin(Error level).
	apiLog  = "api"
	statLog = "stat" // logs statistical data to xxx.stat.log as Error level.
)

// APIType API type
type APIType int

// API type const
const (
	TypeWrite APIType = iota
	TypeRead
	TypeAdmin
)

// Environment variables control Trace, Debug, Info, APIWrite, and APIRead level log output.
// LOG_TRACE=0 disabled Trace level
// LOG_DEBUG=0 disabled Debug level
// LOG_INFO=0 disabled Info level
// LOG_APIWRITE=0 disabled APIWRITE level
// LOG_APIREAD=0 disabled APIREAD level
const (
	logTraceEnv    = "LOG_TRACE"
	logDebugEnv    = "LOG_DEBUG"
	logInfoEnv     = "LOG_INFO"
	logAPIWriteEnv = "LOG_APIWRITE"
	logAPIReadEnv  = "LOG_APIREAD"
)

var (
	traceEnabled    = true
	debugEnabled    = true
	infoEnabled     = true
	apiWriteEnabled = true
	apiReadEnabled  = true
)

// DisableFromEnv Read environment variables `env` to determine whether to enable related level.
// Default is enabled.
// 0 represents disabled, others are enabled.
func DisableFromEnv(env string) bool {
	switch os.Getenv(env) {
	case "0":
		return false
	default:
		return true
	}
}

// KeyValuePair is the wrapper struct of log.WithFields
type KeyValuePair struct {
	Key   string
	Value string
}

/*
// Trace logs to Trace log, Arguments are handled in the manner of key-value pair.
func Trace(pairs ...KeyValuePair) {
	if traceEnabled {
		sweetenPairs(log.Get(normalLog), pairs...).Trace("")
	}
}
*/

// Trace logs to Trace log, Arguments are handled in the manner of fmt.Printf.
func Trace(format string, args ...interface{}) {
	if traceEnabled {
		log.Get(normalLog).Tracef(format, args...)
	}
}

// Tracef logs to Trace log, Arguments are handled in the manner of fmt.Printf.
func Tracef(format string, args ...interface{}) {
	if traceEnabled {
		log.Get(normalLog).Tracef(format, args...)
	}
}

/*
// Debug logs to Debug log, Arguments are handled in the manner of key-value pair.
func Debug(pairs ...KeyValuePair) {
	if debugEnabled {
		sweetenPairs(log.Get(normalLog), pairs...).Debug("")
	}
}
*/

// Debug logs to Debug log, Arguments are handled in the manner of fmt.Printf.
func Debug(format string, args ...interface{}) {
	if debugEnabled {
		log.Get(normalLog).Debugf(format, args...)
	}
}

// Debugf logs to Debug log, Arguments are handled in the manner of fmt.Printf.
func Debugf(format string, args ...interface{}) {
	if debugEnabled {
		log.Get(normalLog).Debugf(format, args...)
	}
}

/*
// Info logs to Info log, Arguments are handled in the manner of key-value pair.
func Info(pairs ...KeyValuePair) {
	if infoEnabled {
		sweetenPairs(log.Get(normalLog), pairs...).Info("")
	}
}
*/

// Info logs to Info log, Arguments are handled in the manner of fmt.Printf.
func Info(format string, args ...interface{}) {
	if infoEnabled {
		log.Get(normalLog).Infof(format, args...)
	}
}

// Infof logs to Info log, Arguments are handled in the manner of fmt.Printf.
func Infof(format string, args ...interface{}) {
	if infoEnabled {
		log.Get(normalLog).Infof(format, args...)
	}
}

/*
// Sys logs to Error log, Arguments are handled in the manner of key-value pair.
func Sys(pairs ...KeyValuePair) {
	sweetenPairs(log.Get(systemLog), pairs...).Info("")
}
*/

// Sys logs to Error log, Arguments are handled in the manner of fmt.Printf.
func Sys(format string, args ...interface{}) {
	log.Get(systemLog).Infof(format, args...)
}

// Sysf logs to Error log, Arguments are handled in the manner of fmt.Printf.
func Sysf(format string, args ...interface{}) {
	log.Get(systemLog).Infof(format, args...)
}

/*
// Warn logs to Warn log, Arguments are handled in the manner of key-value pair.
func Warn(pairs ...KeyValuePair) {
	sweetenPairs(log.Get(wfLog), pairs...).Warn("")
}
*/

// Warn logs to Warn log, Arguments are handled in the manner of fmt.Printf.
func Warn(format string, args ...interface{}) {
	log.Get(wfLog).Warnf(format, args...)
}

// Warnf logs to Warn log, Arguments are handled in the manner of fmt.Printf.
func Warnf(format string, args ...interface{}) {
	log.Get(wfLog).Warnf(format, args...)
}

/*
// Error logs to Error log, Arguments are handled in the manner of key-value pair.
func Error(pairs ...KeyValuePair) {
	sweetenPairs(log.Get(normalLog), pairs...).Error("")
}
*/

// Error logs to Error log, Arguments are handled in the manner of fmt.Printf.
func Error(format string, args ...interface{}) {
	log.Get(normalLog).Errorf(format, args...)
}

// Errorf logs to Error log, Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, args ...interface{}) {
	log.Get(normalLog).Errorf(format, args...)
}

/*
// Fatal logs to Fatal log, Arguments are handled in the manner of key-value pair.
func Fatal(pairs ...KeyValuePair) {
	sweetenPairs(log.Get(wfLog), pairs...).Fatal("")
}
*/

// Fatal logs to Fatal log, Arguments are handled in the manner of fmt.Printf.
func Fatal(format string, args ...interface{}) {
	log.Get(wfLog).Fatalf(format, args...)
}

// Fatalf logs to Fatal log, Arguments are handled in the manner of fmt.Printf.
func Fatalf(format string, args ...interface{}) {
	log.Get(wfLog).Fatalf(format, args...)
}

// API logs according to apiType, Arguments are handled in the manner of fmt.Printf
func API(apiType APIType, apiName string, latencyUs int, errCodeStr string, pairs ...KeyValuePair) {
	l := log.Get(apiLog).WithFields("Api", apiName, "LatencyUs", strconv.Itoa(latencyUs), "ErrCode", errCodeStr)
	l = sweetenPairs(l, pairs...)

	switch apiType {
	case TypeWrite:
		if apiWriteEnabled {
			l.Debug("")
		}
	case TypeRead:
		if apiReadEnabled {
			l.Debug("")
		}
	case TypeAdmin:
		l.Info("")
	}
}

// Stat logs to Error log, Arguments are handled in the manner of fmt.Printf
// It logs statistical data, eg. aggregate data.
func Stat(pairs ...KeyValuePair) {
	sweetenPairs(log.Get(statLog), pairs...).Info("")
}

func sweetenPairs(logger log.Logger, pair ...KeyValuePair) log.Logger {
	if pair != nil {
		for _, v := range pair {
			logger = logger.WithFields(v.Key, v.Value)
		}
	}

	l, ok := logger.(*log.ZapLogWrapper)
	if ok {
		return l.GetLogger()
	}
	return l
}

// Following function with context parameter now is preserved for future TraceID field.
//
// Trace logs to TRACE log, Arguments are handled in the manner of key-value pair.
// func Trace(ctx context.Context, pairs ...KeyValuePair) {
// 	if traceEnabled {
// 		traceID := trpc.GetMetaData(ctx, traceIDKey)
// 		l := log.Get(normalLog).WithFields(traceIDKey, string(traceID))
// 		sweetenPairs(l, pairs...).Trace("")
// 	}
// }
//
// // Tracef logs to TRACE log, Arguments are handled in the manner of fmt.Printf
// func Tracef(ctx context.Context, format string, args ...interface{}) {
// 	if traceEnabled {
// 		traceID := trpc.GetMetaData(ctx, traceIDKey)
// 		log.Get(normalLog).WithFields(traceIDKey, string(traceID),
// 			msgKey, fmt.Sprintf(format, args...)).Trace("")
// 	}
// }
//
// // Debug
// func Debug(ctx context.Context, pairs ...KeyValuePair) {
// 	if debugEnabled {
// 		traceID := trpc.GetMetaData(ctx, traceIDKey)
// 		l := log.Get(normalLog).WithFields(traceIDKey, string(traceID))
// 		sweetenPairs(l, pairs...).Debug("")
// 	}
// }
//
// // Debugf
// func Debugf(ctx context.Context, format string, args ...interface{}) {
// 	if debugEnabled {
// 		traceID := trpc.GetMetaData(ctx, traceIDKey)
// 		log.Get(normalLog).WithFields(traceIDKey, string(traceID),
// 			msgKey, fmt.Sprintf(format, args...)).Debug("")
// 	}
// }
//
// // Info
// func Info(ctx context.Context, pairs ...KeyValuePair) {
// 	if infoEnabled {
// 		traceID := trpc.GetMetaData(ctx, traceIDKey)
// 		l := log.Get(normalLog).WithFields(traceIDKey, string(traceID))
// 		sweetenPairs(l, pairs...).Info("")
// 	}
// }
//
// // Infof logs to Info log, Arguments are handled in the manner of fmt.Printf
// func Infof(ctx context.Context, format string, args ...interface{}) {
// 	if infoEnabled {
// 		traceID := trpc.GetMetaData(ctx, traceIDKey)
// 		log.Get(normalLog).WithFields(traceIDKey, string(traceID),
// 			msgKey, fmt.Sprintf(format, args...)).Info("")
// 	}
// }
//
// // Sys
// func Sys(ctx context.Context, pairs ...KeyValuePair) {
// 	traceID := trpc.GetMetaData(ctx, traceIDKey)
// 	l := log.Get(systemLog).WithFields(traceIDKey, string(traceID))
// 	sweetenPairs(l, pairs...).Error("")
// }
//
// // Sysf logs to Error log, Arguments are handled in the manner of fmt.Printf
// func Sysf(ctx context.Context, format string, args ...interface{}) {
// 	traceID := trpc.GetMetaData(ctx, traceIDKey)
// 	log.Get(systemLog).WithFields(traceIDKey, string(traceID),
// 		msgKey, fmt.Sprintf(format, args...)).Error("")
// }
//
// // Warn logs to Warn log, Arguments are handled in the manner of fmt.Printf
// func Warn(ctx context.Context, pairs ...KeyValuePair) {
// 	traceID := trpc.GetMetaData(ctx, traceIDKey)
// 	l := log.Get(wfLog).WithFields(traceIDKey, string(traceID))
// 	sweetenPairs(l, pairs...).Warn("")
// }
//
// // Error logs to Error log, Arguments are handled in the manner of key-value pair.
// func Error(ctx context.Context, pairs ...KeyValuePair) {
// 	traceID := trpc.GetMetaData(ctx, traceIDKey)
// 	l := log.Get(wfLog).WithFields(traceIDKey, string(traceID))
// 	sweetenPairs(l, pairs...).Error("")
// }
//
// // Fatal logs to Fatal log, Arguments are handled in the manner of fmt.Printf
// func Fatal(ctx context.Context, pairs ...KeyValuePair) {
// 	traceID := trpc.GetMetaData(ctx, traceIDKey)
// 	l := log.Get(wfLog).WithFields(traceIDKey, string(traceID))
// 	sweetenPairs(l, pairs...).Fatal("")
// }
//
// // API logs according to apiType, Arguments are handled in the manner of fmt.Printf
// // TypeWrite logs to Info log
// // TypeRead logs to Debug log
// // TypeAdmin logs yo Error log
// func API(ctx context.Context, apiType APIType, apiName string, delayMs int, errStr, interErrStr string,
// 	pairs ...KeyValuePair) {
// 	traceID := trpc.GetMetaData(ctx, traceIDKey)
// 	l := log.Get(apiLog).WithFields(traceIDKey, string(traceID),
// 		"Api", apiName, "DelayMs", strconv.Itoa(delayMs), "Err", errStr, "InterErr", interErrStr)
// 	l = sweetenPairs(l, pairs...)
//
// 	switch apiType {
// 	case TypeWrite:
// 		if apiWriteEnabled {
// 			l.Info("")
// 		}
// 	case TypeRead:
// 		if apiReadEnabled {
// 			l.Debug("")
// 		}
// 	case TypeAdmin:
// 		l.Error("")
// 	}
// }
//
// // Stat logs to Error log, Arguments are handled in the manner of fmt.Printf
// // It logs statistical data, eg. aggregate data.
// func Stat(pairs ...KeyValuePair) {
// 	sweetenPairs(log.Get(statLog), pairs...).Error("")
// }
