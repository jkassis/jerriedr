package kittie

import (
	context "context"
	"io"
	math "math"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/raft"
	"github.com/jkassis/jerrie/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	// raft metrics
	raftJoinTimeMetrc prometheus.Histogram
	raftPutTimeMetrc  prometheus.Histogram
	raftGetTimeMetrc  prometheus.Histogram

	// serviceGet
	serviceGetReqSizeMetrc prometheus.Histogram
	serviceGetResSizeMetrc prometheus.Histogram
	serviceGetTimeMetrc    prometheus.Histogram

	// servicePut
	servicePutReqSizeMetrc prometheus.Histogram
	servicePutResSizeMetrc prometheus.Histogram
	servicePutTimeMetrc    prometheus.Histogram
)

func init() {
	SizeBuckets := []float64{}
	for i := 0; i < 16; i++ {
		SizeBuckets = append(SizeBuckets, 10*math.Pow(2, float64(i)))
	}
	TimeBuckets := []float64{}
	for i := 0; i < 22; i++ {
		TimeBuckets = append(TimeBuckets, 10*math.Pow(2, float64(i)))
	}

	// RAFT READ
	raftGetTimeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "raft_get_time", Help: "time to read to the service", Buckets: TimeBuckets})
	core.PromRegisterCollector(raftGetTimeMetrc)

	// RAFT WRITE
	raftPutTimeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "raft_put_time", Help: "time to apply in raft", Buckets: TimeBuckets})
	core.PromRegisterCollector(raftPutTimeMetrc)

	// RAFT JOIN
	raftJoinTimeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "raft_join_time", Help: "time to join raft", Buckets: TimeBuckets})
	core.PromRegisterCollector(raftJoinTimeMetrc)

	// SERVICE READ
	serviceGetReqSizeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "service_get_req_size", Help: "size of the read request", Buckets: SizeBuckets})
	core.PromRegisterCollector(serviceGetReqSizeMetrc)
	serviceGetResSizeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "service_get_res_size", Help: "size of the read response", Buckets: SizeBuckets})
	core.PromRegisterCollector(serviceGetResSizeMetrc)
	serviceGetTimeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "service_get_time", Help: "time to read to the service", Buckets: TimeBuckets})
	core.PromRegisterCollector(serviceGetTimeMetrc)

	// SERVICE WRITE
	servicePutReqSizeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "service_put_req_size", Help: "size of the write request", Buckets: SizeBuckets})
	core.PromRegisterCollector(servicePutReqSizeMetrc)
	servicePutResSizeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "service_put_res_size", Help: "size of the write response", Buckets: SizeBuckets})
	core.PromRegisterCollector(servicePutResSizeMetrc)
	servicePutTimeMetrc =
		prometheus.NewHistogram(prometheus.HistogramOpts{Name: "service_put_time", Help: "time to write to the service", Buckets: TimeBuckets})
	core.PromRegisterCollector(servicePutTimeMetrc)
}

// RaftableService is a service that can be rafted with any of the
// raft implementations we might use here.
type RaftableService struct {
	Handler Handler
	Init    func() error
	Play    func() error
	Write   func(ctx context.Context,
		proposalIdx uint64,
		request []byte,
		handler Handler) (handlerRes []byte,
		handlerErr error,
		writeErr error)
	Read                 func(ctx context.Context, request []byte, handler Handler) ([]byte, error)
	SnapshotMake         func() (ServiceRaftSnapshot, error)
	SetLeader            func(bool)
	StatusAliveHandler   Handler
	StatusReadyHandler   Handler
	StatusStartedHandler Handler
	Stop                 func() error
	Open                 func(stopc <-chan struct{}) (uint64, error)
	Sync                 func() error
}

// ServiceRaftSnapshot is a snapshot of a ServiceRaft service
type ServiceRaftSnapshot interface {
	raft.FSMSnapshot
	Restore(source io.ReadCloser) error
	Write(io.Writer) error
	Read(io.Reader) error
	Snap() error
}

// ServiceRaft is a middleman... a service inbetween a transport and a service that does something useful
type ServiceRaft interface {
	ClusterIsJoined() bool
	Init() error
	Play() error
	Stop() error
	Routes() map[string]Handler
	WriteHandler(ctx context.Context, req []byte) ([]byte, error)
	ReadHandler(ctx context.Context, req []byte) ([]byte, error)
	ServerResTimeout() time.Duration
}

// ServiceRaftNode represents all of the pieces of a functional RAFT
type ServiceRaftNode struct {
	BindHost, JoinHostport, AdvertiseHostport string
	Context                                   context.Context
	DataTempDir, DataPermDir, RaftSubDir      string
	HTTPServerResTimeout                      time.Duration
	Name                                      string
	Port                                      int
	RaftInitialMembers                        []string
	ServiceGRPC                               *ServiceGRPC
	ServiceHTTP                               *ServiceHTTP
	ServiceHTTPTLSEnable                      bool
	ServiceHTTPTLSCertFilePath                string
	ServiceHTTPTLSKeyFilePath                 string
	ServiceHTTPFast                           *ServiceHTTPFast
	ServiceNetRPC                             *ServiceNetRPC
	ServiceRaft                               ServiceRaft
	ServiceRaftLocalID                        string
	ServiceRaftRTTMillisecond                 uint64
	ServiceRaftRTTsPerElection                uint64
	ServiceRaftRTTsPerHeartbeat               uint64
	ServiceRaftServerResTimeout               time.Duration
	ServiceRaftSnapshotThreshold              int
	RaftLogLevel                              logrus.Level
	ServiceRaftable                           *RaftableService
}

// Init does initialization
func (s *ServiceRaftNode) Init() (err error) {
	// calculate ports for edge services and raft
	RaftBindHostport := net.JoinHostPort(s.BindHost, strconv.Itoa(s.Port-1))
	HTTPFastHostport := net.JoinHostPort(s.BindHost, strconv.Itoa(s.Port))
	HTTPHostport := net.JoinHostPort(s.BindHost, strconv.Itoa(s.Port+1))
	NetRPCHostport := net.JoinHostPort(s.BindHost, strconv.Itoa(s.Port+2))
	GRPCHostport := net.JoinHostPort(s.BindHost, strconv.Itoa(s.Port+3))

	// create the raft service first...
	// some services (tickie) will need this reference to make raft proposals
	// and we provide the ref during init
	raftLogger := core.LogMake()
	core.Log.Warnf("ServiceRaftNode: RaftLogger: level is '%s'", s.RaftLogLevel.String())
	raftLogger.SetLevel(s.RaftLogLevel)
	raftLogger.SetFormatter(core.LogTextFormatter)

	var useDragonboat bool
	if useDragonboat, err = strconv.ParseBool(os.Getenv("RAFT_USE_DRAGONBOAT")); err != nil {
		useDragonboat = false
	}
	useHashicorp := !useDragonboat
	if useDragonboat {
		core.Log.Warn("ServiceRaftNode: using dragonboat")
		s.ServiceRaft = &ServiceRaftDBDisk{
			Context:               s.Context,
			RaftAdvertiseHostport: s.AdvertiseHostport,
			RaftBindHostport:      RaftBindHostport,
			RaftInitialMembers:    s.RaftInitialMembers,
			RaftJoinHostport:      s.JoinHostport,
			RaftLocalID:           s.ServiceRaftLocalID,
			RaftLogger:            raftLogger,
			RaftNodeHostDir:       s.DataPermDir + s.RaftSubDir + "/dragonboat/nodehost",
			RaftRTTMillisecond:    s.ServiceRaftRTTMillisecond,
			RaftRTTsPerElection:   s.ServiceRaftRTTsPerElection,
			RaftRTTsPerHeartbeat:  s.ServiceRaftRTTsPerHeartbeat,
			RaftServerResTimeout:  s.ServiceRaftServerResTimeout,
			RaftSnapshotDir:       s.DataPermDir + s.RaftSubDir + "/dragonboat/snapshots",
			RaftSnapshotThreshold: s.ServiceRaftSnapshotThreshold,
			RaftSnapshotsToRetain: 3,
			RaftWALDir:            s.DataPermDir + "/raft/dragonboat/wal",
			RaftableService:       s.ServiceRaftable,
		}
	}

	if useHashicorp {
		core.Log.Warn("ServiceRaftNode: using hashicorp")
		s.ServiceRaft = &ServiceRaftHC{
			Context:               s.Context,
			RaftAdvertiseHostport: s.AdvertiseHostport,
			RaftBindHostport:      RaftBindHostport,
			RaftDBDir:             s.DataPermDir + s.RaftSubDir + "/hashicorp/db",
			RaftInitialMembers:    s.RaftInitialMembers,
			RaftJoinHostport:      s.JoinHostport,
			RaftLocalID:           s.ServiceRaftLocalID,
			RaftLogger:            raftLogger,
			RaftServerResTimeout:  s.ServiceRaftServerResTimeout,
			RaftSnapshotDir:       s.DataPermDir + s.RaftSubDir + "/hashicorp/snapshots",
			RaftSnapshotThreshold: s.ServiceRaftSnapshotThreshold,
			RaftSnapshotsToRetain: 3,
			RaftableService:       s.ServiceRaftable,
		}
	}

	// init the service without error before allowing traffic
	if err := s.ServiceRaftable.Init(); err != nil {
		return err
	}

	// create the edge services
	// ServiceHTTPFast
	serviceHTTPFast := &ServiceHTTPFast{
		Addr:                 HTTPFastHostport,
		Context:              s.Context,
		HTTPServerResTimeout: s.HTTPServerResTimeout,
		TLSEnable:            s.ServiceHTTPTLSEnable,
		TLSCertFilePath:      s.ServiceHTTPTLSCertFilePath,
		TLSKeyFilePath:       s.ServiceHTTPTLSKeyFilePath,
	}
	serviceHTTPFast.Init()
	serviceHTTPFast.RouteAdd("/service", s.ServiceRaftable.Handler)
	serviceHTTPFast.RouteAdd("/statusAlive", s.ServiceRaftable.StatusAliveHandler)
	serviceHTTPFast.RouteAdd("/statusReady", s.ServiceRaftable.StatusReadyHandler)
	serviceHTTPFast.RouteAdd("/statusStarted", s.ServiceRaftable.StatusStartedHandler)
	s.ServiceHTTPFast = serviceHTTPFast

	// ServiceHTTP
	serviceHTTP := &ServiceHTTP{
		Addr:                 HTTPHostport,
		Context:              s.Context,
		HTTPServerResTimeout: s.HTTPServerResTimeout,
		ShutdownGracePeriod:  ServiceShutdownGracePeriodGet(),
		TLSEnable:            s.ServiceHTTPTLSEnable,
		TLSCertFilePath:      s.ServiceHTTPTLSCertFilePath,
		TLSKeyFilePath:       s.ServiceHTTPTLSKeyFilePath,
	}
	serviceHTTP.Init()
	serviceHTTP.RouteAdd("service", s.ServiceRaftable.Handler)
	serviceHTTP.RouteAdd("statusAlive", s.ServiceRaftable.StatusAliveHandler)
	serviceHTTP.RouteAdd("statusReady", s.ServiceRaftable.StatusReadyHandler)
	serviceHTTP.RouteAdd("statusStarted", s.ServiceRaftable.StatusStartedHandler)
	s.ServiceHTTP = serviceHTTP

	// ServiceNetRPC
	serviceNetRPC := &ServiceNetRPC{
		Addr:    NetRPCHostport,
		Context: s.Context,
	}
	serviceNetRPC.Init()
	serviceNetRPC.RouteAdd("service", s.ServiceRaftable.Handler)
	serviceNetRPC.RouteAdd("statusAlive", s.ServiceRaftable.StatusAliveHandler)
	serviceNetRPC.RouteAdd("statusReady", s.ServiceRaftable.StatusReadyHandler)
	serviceNetRPC.RouteAdd("statusStarted", s.ServiceRaftable.StatusStartedHandler)
	s.ServiceNetRPC = serviceNetRPC

	// ServiceGRPC
	serviceGRPC := &ServiceGRPC{
		Addr: GRPCHostport,
	}
	serviceGRPC.Init()
	s.ServiceGRPC = serviceGRPC

	// add raft service routes to all edge services
	for route, handler := range s.ServiceRaft.Routes() {
		serviceHTTPFast.RouteAdd("/"+route, handler)
		serviceHTTP.RouteAdd(route, handler)
		serviceNetRPC.RouteAdd(route, handler)
		// serviceGRPC.RouteAdd(route, handler)
	}

	// init the raft last...
	// there will be WAL entries waiting to play against the service...
	if err := s.ServiceRaft.Init(); err != nil {
		core.Log.Fatal(err)
	}

	return nil
}

// Play runs a server as an embedded go routine
func (s *ServiceRaftNode) Play() (err error) {
	if err := s.ServiceRaftable.Play(); err != nil {
		core.Log.Fatal(err)
	}

	if err := s.ServiceRaft.Play(); err != nil {
		core.Log.Fatal(err)
	}

	if err := s.ServiceHTTPFast.Play(); err != nil {
		core.Log.Fatal(err)
	}

	if err := s.ServiceHTTP.Play(); err != nil {
		core.Log.Fatal(err)
	}

	s.ServiceNetRPC.Play()

	if err := s.ServiceGRPC.Play(); err != nil {
		core.Log.Fatal(err)
	}

	// Init and play
	return nil
}

// Stop stops a stack
func (s *ServiceRaftNode) Stop(ctx context.Context) (err error) {
	// Stop the fast http service
	err = s.ServiceHTTPFast.Stop()
	if err != nil {
		return err
	}

	err = s.ServiceGRPC.Stop()
	if err != nil {
		return err
	}

	err = s.ServiceNetRPC.Stop()
	if err != nil {
		return err
	}

	err = s.ServiceHTTP.Stop()
	if err != nil {
		return err
	}

	err = s.ServiceRaft.Stop()
	if err != nil {
		return err
	}

	err = s.ServiceRaftable.Stop()
	if err != nil {
		return err
	}
	return nil
}
