package logrus

import (
	"bytes"
	"io"
	"log"
	"os"

	"github.com/hashicorp/go-hclog"
)

// AdapterHCLog implements the hclog interface, and wraps it
// around a Logrus entry
type AdapterHCLog struct {
	MyLogger FieldLogger
	MyName   string
}

// HCLog has one more level than we do. As such, we will never
// set trace level.
func (*AdapterHCLog) Trace(_ string, _ ...interface{}) {
	return
}

func (a *AdapterHCLog) Debug(msg string, args ...interface{}) {
	a.CreateEntry(args).Debug(msg)
}

func (a *AdapterHCLog) Info(msg string, args ...interface{}) {
	a.CreateEntry(args).Info(msg)
}

func (a *AdapterHCLog) Warn(msg string, args ...interface{}) {
	a.CreateEntry(args).Warn(msg)
}

func (a *AdapterHCLog) Error(msg string, args ...interface{}) {
	a.CreateEntry(args).Error(msg)
}

func (a *AdapterHCLog) IsTrace() bool {
	return false
}

func (a *AdapterHCLog) IsDebug() bool {
	return a.shouldEmit(DebugLevel)
}

func (a *AdapterHCLog) IsInfo() bool {
	return a.shouldEmit(InfoLevel)
}

func (a *AdapterHCLog) IsWarn() bool {
	return a.shouldEmit(WarnLevel)
}

func (a *AdapterHCLog) IsError() bool {
	return a.shouldEmit(ErrorLevel)
}

func (a *AdapterHCLog) SetLevel(hclog.Level) {
	// interface definition says it is ok for this to be a noop if
	// implementations don't need/want to support dynamic level changing, which
	// we don't currently.
}

func (a *AdapterHCLog) With(args ...interface{}) hclog.Logger {
	e := a.CreateEntry(args)
	return &AdapterHCLog{MyLogger: e}
}

func (a *AdapterHCLog) ImpliedArgs() []interface{} {
	return nil
}

func (a *AdapterHCLog) Name() string {
	return a.MyName
}

func (a *AdapterHCLog) Named(name string) hclog.Logger {
	var newName bytes.Buffer
	if a.MyName != "" {
		newName.WriteString(a.MyName)
		newName.WriteString(".")
	}
	newName.WriteString(name)

	return a.ResetNamed(newName.String())
}

func (a *AdapterHCLog) ResetNamed(name string) hclog.Logger {
	fields := []interface{}{"subsystem_name", name}
	e := a.CreateEntry(fields)
	return &AdapterHCLog{MyLogger: e, MyName: name}
}

// StandardLogger is meant to return a stdlib Logger type which wraps around
// hclog. It does this by providing an io.Writer and instantiating a new
// Logger. It then tries to interpret the log level by parsing the message.
//
// Since we are not using `hclog` in a generic way, and I cannot find any
// calls to this method from go-plugin, we will poorly support this method.
// Rather than pull in all of hclog writer parsing logic, pass it a Logrus
// writer, and hardcode the level to INFO.
//
// Apologies to those who find themselves here.
func (a *AdapterHCLog) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	entry := a.MyLogger.WithFields(Fields{})
	return log.New(entry.WriterLevel(InfoLevel), "", 0)
}

func (a *AdapterHCLog) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	var w io.Writer
	logger, ok := a.MyLogger.(*Logger)
	if ok {
		w = logger.Out
	}
	if w == nil {
		w = os.Stderr
	}
	return w
}

func (a *AdapterHCLog) shouldEmit(level Level) bool {
	currentLevel := a.MyLogger.WithFields(Fields{}).Level
	if currentLevel >= level {
		return true
	}

	return false
}

func (a *AdapterHCLog) CreateEntry(args []interface{}) *Entry {
	if len(args)%2 != 0 {
		args = append(args, "<unknown>")
	}

	fields := make(Fields)
	for i := 0; i < len(args); i = i + 2 {
		k, ok := args[i].(string)
		if !ok {
		}
		v := args[i+1]
		fields[k] = v
	}

	return a.MyLogger.WithFields(fields)
}

func (a *AdapterHCLog) Log(level hclog.Level, msg string, args ...interface{}) {
	if level == hclog.Off {
		return
	}
	a.CreateEntry(args).Log(hcLevelToLogrusLevel[level], msg)
}

var hcLevelToLogrusLevel = map[hclog.Level]Level{
	hclog.NoLevel: WarnLevel,
	hclog.Error:   ErrorLevel,
	hclog.Warn:    WarnLevel,
	hclog.Info:    InfoLevel,
	hclog.Debug:   DebugLevel,
	hclog.Trace:   TraceLevel,
}
