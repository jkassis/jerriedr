package kittie

import (
	"github.com/jkassis/jerrie/core"
	"github.com/lni/dragonboat/v3/logger"
	"github.com/sirupsen/logrus"
)

// DefaultClusterID The default cluster ID expressed as a UUID
const DefaultClusterID uint64 = 10

// JoinRequestData data for a join request
type JoinRequestData struct {
	Hostport  string
	NodeID    uint64
	ClusterID uint64
}

const (
	smResultSuccess = iota
	smResultError   = iota
)

func init() {
	logCache = make(map[string]*DBLogrusLogger)
	logger.SetLoggerFactory(logFactory)

	// These SetLevel calls don't make a difference
	logger.GetLogger("raft").SetLevel(logger.DEBUG)
	logger.GetLogger("rsm").SetLevel(logger.DEBUG)
	logger.GetLogger("transport").SetLevel(logger.DEBUG)
	logger.GetLogger("grpc").SetLevel(logger.DEBUG)
}

// DBLogrusLogger adapter
type DBLogrusLogger struct{ log *logrus.Logger }

// SetLevel wrapper
func (d DBLogrusLogger) SetLevel(level logger.LogLevel) { d.log.SetLevel(logrus.Level(uint32(level))) }

// Debugf wrapper
func (d DBLogrusLogger) Debugf(format string, args ...interface{}) { d.log.Debugf(format, args...) }

// Infof wrapper
func (d DBLogrusLogger) Infof(format string, args ...interface{}) { d.log.Infof(format, args...) }

// Warningf wrapper
func (d DBLogrusLogger) Warningf(format string, args ...interface{}) { d.log.Warningf(format, args...) }

// Errorf wrapper
func (d DBLogrusLogger) Errorf(format string, args ...interface{}) { d.log.Errorf(format, args...) }

// Panicf wrapper
func (d DBLogrusLogger) Panicf(format string, args ...interface{}) { d.log.Panicf(format, args...) }

var logCache map[string]*DBLogrusLogger

func logFactory(pkgName string) logger.ILogger {
	dbLog, ok := logCache[pkgName]
	if !ok {
		dbLog = &DBLogrusLogger{log: core.LogMake()}
		logCache[pkgName] = dbLog
	}

	return dbLog
}

// DBRaftProposalIDXK is the key for KV database entries meant to track this.
var DBRaftProposalIDXK = &core.DBInt64K{Key: "DBRaftProposalIDX"}
