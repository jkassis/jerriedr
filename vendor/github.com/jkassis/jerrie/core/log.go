package core

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"time"

	"cloud.google.com/go/profiler"
	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
)

// Log global logger
var Log *logrus.Logger

// LogTextFormatter is a good choice for debug mode
var LogTextFormatter = &logrus.TextFormatter{
	DisableColors: false,
	CallerPrettyfier: func(f *runtime.Frame) (string, string) {
		file := f.File
		if len(file) > 33 {
			file = file[len(file)-33:]
		}
		return "", fmt.Sprintf(" %s %.33s:%4.d ", time.Now().Format(time.StampNano), file, f.Line)
	},
}

// LogJSONFormatter is justa global instance of logrus.JSONFormatter
var LogJSONFormatter = new(logrus.JSONFormatter)

// LogNullWriter writes to nowhere
type LogNullWriter struct{}

func (nullWriter *LogNullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// Init inits the core log service using LOG_LEVEL environment variable
var sentryEnabled bool

func init() {
	if Log != nil {
		return
	}

	Log = LogMake()
	tm := time.Now()
	Log.Warn("Logging system started " + tm.String())

	// Init Sentry for crash reporting
	sentryEnabled = os.Getenv("SENTRY_ENABLED") == "TRUE"
	if sentryEnabled {
		sentryTransport := sentry.NewHTTPTransport()
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:       os.Getenv("SENTRY_DSN"),
			Transport: sentryTransport,
		}); err != nil {
			Log.Fatal(err)
		}
	}

	// StackDriver Profiling
	googleProfilerEnabled := os.Getenv("GOOGLE_PROFILER_ENABLED")
	if googleProfilerEnabled != "TRUE" {
		Log.Warnf("Profiling disabled.")
	} else {
		googleProjectID := os.Getenv("GOOGLE_PROFILER_PROJECT_ID")
		if googleProjectID == "" {
			Log.Warnf("Did not find a googleProjectID. Profiling disabled.")
		} else {
			Log.Warnf("Profiling enabled.")
			if err := profiler.Start(profiler.Config{
				Service:              "multi",
				NoHeapProfiling:      false,
				NoAllocProfiling:     false,
				MutexProfiling:       true,
				NoGoroutineProfiling: false,
				DebugLogging:         true,
				ProjectID:            googleProjectID,
			}); err != nil {
				Log.Fatal(err)
			}
		}
	}
}

// LogMake returns a new logger
func LogMake() (logger *logrus.Logger) {
	format := os.Getenv(("LOG_FORMAT"))
	var formatter logrus.Formatter
	if format == "json" {
		formatter = LogJSONFormatter
	} else {
		formatter = LogTextFormatter
	}
	// Set log level
	logrusLogger := &logrus.Logger{Out: os.Stderr,
		Formatter: formatter,
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.WarnLevel}
	logrusLogger.SetReportCaller(true)
	logrusLogger.Debug("Starting...")
	serviceLogLevel := os.Getenv("LOG_LEVEL")
	level, err := logrus.ParseLevel(serviceLogLevel)
	if err != nil {
		logrusLogger.Panic("LOG_LEVEL Invalid")
		panic(err)
	}
	logrusLogger.SetLevel(level)
	logrusLogger.Debug("LOG_LEVEL=" + serviceLogLevel)
	return logrusLogger
}

// URLPath gets just the path portion of a url
func URLPath(rawurl string) string {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	return parsedURL.Path
}

// SentryRecover returns a function that reports panics to Sentry and recovers
func SentryRecover(context string) {
	err := recover()
	if sentryEnabled && err != nil {
		localHub := sentry.CurrentHub().Clone()
		localHub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("context", context)
		})
		localHub.Recover(err)
		localHub.Flush(time.Second * 5)
	}
}
