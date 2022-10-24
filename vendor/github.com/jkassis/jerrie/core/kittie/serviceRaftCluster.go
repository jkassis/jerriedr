package kittie

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/profiler"
	"github.com/jkassis/jerrie/core"
	"github.com/sirupsen/logrus"
)

// ServiceRaftCluster represents all of the pieces of a functional RAFT cluster
type ServiceRaftCluster struct {
	DataTempDir, DataPermDir, RaftSubDir    string
	ForkChildren                            bool
	ForkCmds                                []*exec.Cmd
	GremlinsEnabled                         bool
	HTTPServerResTimeout                    time.Duration
	InitialMembers                          string
	Name, BindHost, JoinHostport            string
	NodeStartIndex, NodeCount, BindPortBase int
	Nodes                                   []*ServiceRaftNode
	ProfilingEnabled                        bool
	ProfilingProjectID                      string
	RaftLogLevel                            string
	RaftRTTMillisecond                      uint64
	RaftRTTsPerElection                     uint64
	RaftRTTsPerHeartbeat                    uint64
	RaftServerResTimeout                    time.Duration
	RaftableServiceLauncher                 func(int, []string) (*exec.Cmd, error)
	RaftableServiceMaker                    func(clusterNodeIdx int, clusterNodeCnt int, serviceDataPermDir string, serviceDataTempDir string, serviceRaftNode *ServiceRaftNode) *RaftableService
	ServiceRaftLogrusLevel                  logrus.Level
	ServiceHTTPTLSEnable                    bool
	ServiceHTTPTLSCertFilePath              string
	ServiceHTTPTLSKeyFilePath               string
	SingleHostCluster                       bool
	SnapshotThreshold                       int
}

// ClusterIsJoined returns true if each node has joined a RAFT cluster
// Note that this does not mean they have joined the same RAFT cluster. Which cluster they
// joine is a fuction of the JOIN Address
func (s *ServiceRaftCluster) ClusterIsJoined() bool {
	for _, node := range s.Nodes {
		if !node.ServiceRaft.ClusterIsJoined() {
			return false
		}
	}
	return true
}

// Init initializes the ServiceRaftStack
func (s *ServiceRaftCluster) Init() {
	n := core.MaxInt(s.NodeCount, (s.NodeStartIndex + 1))
	core.Log.Warn("Initing with room for ", n)
	s.Nodes = make([]*ServiceRaftNode, n)
	s.ForkCmds = make([]*exec.Cmd, n)

	if s.ProfilingEnabled {
		// https://cloud.google.com/profiler/docs/about-profiler
		core.Log.Warn("Starting StackDriver Profiler with Project ID : " + s.ProfilingProjectID)
		err := profiler.Start(profiler.Config{
			Service:              s.Name,
			NoHeapProfiling:      false,
			NoAllocProfiling:     false,
			NoGoroutineProfiling: false,
			DebugLogging:         false,
			ProjectID:            s.ProfilingProjectID,
		})
		if err != nil {
			panic(err)
		}
	}

	level, err := logrus.ParseLevel(s.RaftLogLevel)
	if err != nil {
		panic(fmt.Errorf("Error parsing RaftLogLevel: %w", err))
	}
	s.ServiceRaftLogrusLevel = level
}

// FlagsRegister declares all import flags for ServiceCluster but does not trigger parsing
func (s *ServiceRaftCluster) FlagsRegister() {
	var err error

	// serviceForkChildren tells us to start proc in separate processes
	s.ForkChildren, _ = strconv.ParseBool(os.Getenv("SERVICE_FORK_CHILDREN"))
	flag.BoolVar(&s.ForkChildren, "ForkChildren", s.ForkChildren, "Separate cluster node processes")

	// ServiceDataTempDir from env, default, or flag
	ServiceDataTempDir := os.Getenv("SERVICE_DATA_TEMP_DIR")
	if ServiceDataTempDir == "" {
		ServiceDataTempDir = "/tmp/" + s.Name
	}
	flag.StringVar(&s.DataTempDir, "DataTempDir", ServiceDataTempDir, "Set the HTTP bind address")

	// ServiceDataPermDir from env, default, or flag
	ServiceDataPermDir := os.Getenv("SERVICE_DATA_PERM_DIR")
	if ServiceDataPermDir == "" {
		ServiceDataPermDir = "/var/" + s.Name
	}
	flag.StringVar(&s.DataPermDir, "DataPermDir", ServiceDataPermDir, "Set the HTTP bind address")

	// ServiceRaftSubDir from env, default, or flag
	ServiceRaftSubDir := os.Getenv("SERVICE_RAFT_SUB_DIR")
	if ServiceRaftSubDir == "" {
		ServiceRaftSubDir = "/raft"
	}
	flag.StringVar(&s.RaftSubDir, "RaftSubDir", ServiceRaftSubDir, "The subdirectory of the permDir to use for raft data")

	// ServiceGremlins
	s.GremlinsEnabled, _ = strconv.ParseBool(os.Getenv("SERVICE_GREMLINS"))
	flag.BoolVar(&s.GremlinsEnabled, "Gremlins", s.GremlinsEnabled, "Service gremlins")

	// ServiceNodeCount from environment
	ServiceNodeCount, _ := strconv.Atoi(os.Getenv("SERVICE_NODE_COUNT"))
	flag.IntVar(&s.NodeCount, "NodeCount", ServiceNodeCount, "Service Node Count")
	if ServiceNodeCount == 0 {
		ServiceNodeCount = 1
	}

	// ServiceStartIndex
	ServiceNodeStartIndex, _ := strconv.Atoi(os.Getenv("SERVICE_NODE_START_INDEX"))
	flag.IntVar(&s.NodeStartIndex, "NodeStartIndex", ServiceNodeStartIndex, "Service Node Start Index")

	// ServiceHost
	ServiceBindHost := os.Getenv("SERVICE_BIND_HOST")
	flag.StringVar(&s.BindHost, "BindHost", ServiceBindHost, "Hostname for this server to BIND")

	// ServiceBindPortBase
	ServiceHTTPBindPortBase, _ := strconv.Atoi(os.Getenv("SERVICE_BIND_PORT_BASE"))
	if ServiceHTTPBindPortBase == 0 {
		ServiceHTTPBindPortBase = 4500
	}
	flag.IntVar(&s.BindPortBase, "BindPortBase", ServiceHTTPBindPortBase, "Port for HTTP service")

	// ServiceProfilingProjectID from env, default, or flag
	ServiceProfilingProjectID := os.Getenv("GOOGLE_PROFILER_PROJECT_ID")
	flag.StringVar(&s.ProfilingProjectID, "ProfilingProjectID", ServiceProfilingProjectID, "ProjectID for the profiling service.")

	// ServiceProfilingEnabled from env, default, or flag
	ServiceProfilingEnabled, _ := strconv.ParseBool(os.Getenv("GOOGLE_PROFILER_ENABLED"))
	flag.BoolVar(&s.ProfilingEnabled, "ProfilingEnabled", ServiceProfilingEnabled, "Enable export of profiling data (currently StackDriver)")

	// ServiceRaftJoinHostport from env, default, or flag
	ServiceRaftJoinHostport := os.Getenv("SERVICE_JOIN_HOSTPORT")
	flag.StringVar(&s.JoinHostport, "JoinHostport", ServiceRaftJoinHostport, "Addr for RAFT Join")

	// ServiceHTTPTLSEnable from env, default, or flag
	ServiceHTTPTLSEnable, _ := strconv.ParseBool(os.Getenv("SERVICE_HTTP_TLS_ENABLE"))
	flag.BoolVar(&s.ServiceHTTPTLSEnable, "ServiceHTTPTLSEnable", ServiceHTTPTLSEnable, "HTTP Service TLS Enable")

	// ServiceRaftJoinHostport from env, default, or flag
	ServiceHTTPTLSCertFilePath := os.Getenv("SERVICE_HTTP_TLS_CERT_FILE_PATH")
	flag.StringVar(&s.ServiceHTTPTLSCertFilePath, "ServiceHTTPTLSCertFilePath", ServiceHTTPTLSCertFilePath, "HTTP Service Cert File Path")

	// ServiceRaftJoinHostport from env, default, or flag
	ServiceHTTPTLSKeyFilePath := os.Getenv("SERVICE_HTTP_TLS_KEY_FILE_PATH")
	flag.StringVar(&s.ServiceHTTPTLSKeyFilePath, "ServiceHTTPTLSKeyFilePath", ServiceHTTPTLSKeyFilePath, "HTTP Service Key File Path")

	// ServiceRaftJoinHostport from env, default, or flag
	ServiceRaftInitialMembers := os.Getenv("SERVICE_RAFT_INITIAL_MEMBERS")
	flag.StringVar(&s.InitialMembers, "InitialMembers", ServiceRaftInitialMembers, "Initial member hostports")

	// SingleHostCluster from env, default, or flag
	SingleHostCluster, _ := strconv.ParseBool(os.Getenv("SINGLE_HOST_CLUSTER"))
	flag.BoolVar(&s.SingleHostCluster, "SingleHostCluster", SingleHostCluster, "Running cluster on a single host")

	// RaftLogLevel
	RaftLogLevel := os.Getenv("RAFT_LOG_LEVEL")
	flag.StringVar(&s.RaftLogLevel, "RaftLogLevel", RaftLogLevel, "Service Raft Log Level")

	// ServiceStartIndex
	ServiceRaftSnapshotThreshold, _ := strconv.Atoi(os.Getenv("SERVICE_RAFT_SNAPSHOT_THRESHOLD"))
	flag.IntVar(&s.SnapshotThreshold, "SnapshotThreshold", ServiceRaftSnapshotThreshold, "Service Raft Snapshot Threshold")

	// HTTPServerResTimeout
	HTTPServerResTimeout, err := time.ParseDuration(os.Getenv("HTTP_SERVER_RES_TIMEOUT"))
	if err != nil {
		core.Log.Fatal("could not parse HTTP_SERVER_RES_TIMEOUT env var", err)
	}
	flag.DurationVar(&s.HTTPServerResTimeout, "HTTPServerResTimeout", HTTPServerResTimeout, "Request timeout in a form that can be parsed by golang time.ParseDuration")

	// RaftRTTMillisecond
	RaftRTTMillisecond, err := strconv.ParseUint(os.Getenv("RAFT_SERVER_RTT_MILLIS"), 10, 64)
	if err != nil {
		core.Log.Fatal("could not parse RAFT_SERVER_RTT_MILLIS env var", err)
	}
	flag.Uint64Var(&s.RaftRTTMillisecond, "RaftRTTMillisecond", RaftRTTMillisecond, "Raft proposal RTT in milliseconds.")

	// RaftRTTsPerHeartbeat
	RaftRTTsPerHeartbeat, err := strconv.ParseUint(os.Getenv("RAFT_SERVER_RTTs_PER_HEARTBEAT"), 10, 64)
	if err != nil {
		core.Log.Fatal("could not parse RAFT_SERVER_RTTs_PER_HEARTBEAT env var", err)
	}
	flag.Uint64Var(&s.RaftRTTsPerHeartbeat, "RaftRTTsPerHeartbeat", RaftRTTsPerHeartbeat, "Raft proposal RTT in milliseconds.")

	// RaftRTTsPerSnapshot
	RaftRTTsPerElection, err := strconv.ParseUint(os.Getenv("RAFT_SERVER_RTTs_PER_ELECTION"), 10, 64)
	if err != nil {
		core.Log.Fatal("could not parse RAFT_SERVER_RTTs_PER_ELECTION env var", err)
	}
	flag.Uint64Var(&s.RaftRTTsPerElection, "RaftRTTsPerElection", RaftRTTsPerElection, "Raft proposal RTTs per election.")

	// RaftServerResTimeout
	RaftServerResTimeout, err := time.ParseDuration(os.Getenv("RAFT_SERVER_RES_TIMEOUT"))
	if err != nil {
		core.Log.Fatal("could not parse RAFT_SERVER_RES_TIMEOUT env var", err)
	}
	flag.DurationVar(&s.RaftServerResTimeout, "RaftServerResTimeout", RaftServerResTimeout, "Raft proposal timeout in a form that can be parsed by golang time.ParseDuration")

	// PARSE ARGUMENTS
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <raft-data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

// ForkOne creates a node via fork
func (s *ServiceRaftCluster) ForkOne(clusterNodeIndex int) (*exec.Cmd, error) {
	return s.RaftableServiceLauncher(clusterNodeIndex, s.CLIArgsGet(clusterNodeIndex))
}

// CLIArgsGet returns command line arguments used when forking children
func (s *ServiceRaftCluster) CLIArgsGet(clusterNodeIndex int) []string {
	args := []string{
		"-BindHost=" + s.BindHost,
		"-BindPortBase=" + strconv.Itoa(s.BindPortBase),
		"-DataPermDir=" + s.DataPermDir,
		"-DataTempDir=" + s.DataTempDir,
		"-ForkChildren=false", // This is not a mistake. Prevents recursive forking.
		"-HTTPServerResTimeout=" + s.HTTPServerResTimeout.String(),
		"-InitialMembers=" + s.InitialMembers,
		"-JoinHostport=" + s.JoinHostport,
		"-NodeCount=1",
		"-NodeStartIndex=" + strconv.Itoa(clusterNodeIndex),
		"-ProfilingEnabled=" + strconv.FormatBool(s.ProfilingEnabled),
		"-ProfilingProjectID=" + s.ProfilingProjectID,
		"-RaftLogLevel=" + s.RaftLogLevel,
		"-RaftRTTMillisecond=" + strconv.FormatUint(s.RaftRTTMillisecond, 10),
		"-RaftRTTsPerElection=" + strconv.FormatUint(s.RaftRTTsPerElection, 10),
		"-RaftRTTsPerHeartbeat=" + strconv.FormatUint(s.RaftRTTsPerHeartbeat, 10),
		"-RaftServerResTimeout=" + s.RaftServerResTimeout.String(),
		"-RaftSubDir=" + s.RaftSubDir,
		"-ServiceHTTPTLSCertFilePath=" + s.ServiceHTTPTLSCertFilePath,
		"-ServiceHTTPTLSEnable=" + strconv.FormatBool(s.ServiceHTTPTLSEnable),
		"-ServiceHTTPTLSKeyFilePath=" + s.ServiceHTTPTLSKeyFilePath,
		"-SingleHostCluster=" + strconv.FormatBool(s.SingleHostCluster),
		"-SnapshotThreshold=" + strconv.Itoa(s.SnapshotThreshold),
	}

	return args
}

// Play starts the stack
func (s *ServiceRaftCluster) Play() error {
	if s.ForkChildren {
		// Start the cluster
		for i := s.NodeStartIndex; i < s.NodeStartIndex+s.NodeCount; i++ {
			forkCmd, err := s.ForkOne(i)
			if err != nil {
				return err
			}
			s.ForkCmds[i] = forkCmd
			time.Sleep(15 * time.Second)
		}

		// Play Gremlins
		if s.GremlinsEnabled {
			go s.PlayGremlins()
		}

		// Wait for OS Signal
		ServiceWaitForOSSignal()

		// Killall now
		for _, nodeCmd := range s.ForkCmds {
			if nodeCmd == nil {
				continue
			}
			// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
			err := syscall.Kill(-nodeCmd.Process.Pid, syscall.SIGKILL)
			if err != nil {
				core.Log.Error(err)
			}
		}
	} else {
		// Start the cluster sequentially
		startupChan := make(chan struct{})
		for i := s.NodeStartIndex; i < s.NodeStartIndex+s.NodeCount; i++ {
			go func() {
				clusterNodeIdx := i
				core.Log.Warn("Playing ServiceNode with index ", clusterNodeIdx)
				_, err := s.PlayOne(clusterNodeIdx, s.RaftableServiceMaker)
				if err != nil {
					core.Log.Fatal(err)
				}
				startupChan <- struct{}{}
			}()
			<-startupChan
		}

		// technically... this is the receiving AND sending end for the channel...
		// since the sending code is right up there ^^^
		close(startupChan)

		// Play Gremlins
		if s.GremlinsEnabled {
			go s.PlayGremlins()
		}

		// Wait for OS Signal
		ServiceWaitForOSSignal()

		// Kill sub servers with 10 second shutdown timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		for _, stack := range s.Nodes {
			stack.Stop(ctx)
		}
	}

	return nil
}

// PlayOne runs a server as an embedded go routine
func (s *ServiceRaftCluster) PlayOne(clusterNodeIdx int,
	RaftableServiceMaker func(
		clusterNodeIdx int,
		clusterNodeCnt int,
		serviceDataPermDir string,
		serviceDataTempDir string,
		serviceRaftNode *ServiceRaftNode) *RaftableService) (serviceRaftNode *ServiceRaftNode,
	err error) {
	// join address
	var hostname string
	if s.SingleHostCluster {
		hostname = "local"
	} else {
		hostname, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}
	serviceNodeID := hostname + "-server-" + strconv.Itoa(clusterNodeIdx)
	serviceDataTempDir := s.DataTempDir + "/" + serviceNodeID

	// remove everything from the temp dir!
	// Raft FSMs are not inherently persistent. RAFT restores them by loading a snapshot
	// and then playing raft log history. The temporary directory is passed to the service
	// so that the FSM can use the file system for temporary storage. That's useful for FSMs
	// whose data models exceed the limits of available RAM for example.
	os.RemoveAll(serviceDataTempDir)
	os.MkdirAll(serviceDataTempDir, os.ModeDir|0777)

	// they also need a DataPermDir for raft logs and snapshots
	serviceDataPermDir := s.DataPermDir + "/" + serviceNodeID

	// Service
	AdvertiseHost := os.Getenv("SERVICE_RAFT_ADVERTISE_HOST")

	s.Nodes[clusterNodeIdx] = &ServiceRaftNode{
		AdvertiseHostport:            AdvertiseHost + ":" + strconv.Itoa(s.BindPortBase+100*clusterNodeIdx-1),
		BindHost:                     s.BindHost,
		Context:                      context.Background(),
		DataPermDir:                  serviceDataPermDir,
		DataTempDir:                  serviceDataTempDir,
		RaftSubDir:                   s.RaftSubDir,
		HTTPServerResTimeout:         s.HTTPServerResTimeout,
		JoinHostport:                 s.JoinHostport,
		Name:                         s.Name,
		Port:                         s.BindPortBase + 100*clusterNodeIdx,
		RaftInitialMembers:           strings.Split(s.InitialMembers, ","),
		RaftLogLevel:                 s.ServiceRaftLogrusLevel,
		ServiceRaftLocalID:           serviceNodeID,
		ServiceRaftRTTMillisecond:    s.RaftRTTMillisecond,
		ServiceRaftRTTsPerElection:   s.RaftRTTsPerElection,
		ServiceRaftRTTsPerHeartbeat:  s.RaftRTTsPerHeartbeat,
		ServiceRaftServerResTimeout:  s.RaftServerResTimeout,
		ServiceRaftSnapshotThreshold: s.SnapshotThreshold,
		ServiceHTTPTLSEnable:         s.ServiceHTTPTLSEnable,
		ServiceHTTPTLSCertFilePath:   s.ServiceHTTPTLSCertFilePath,
		ServiceHTTPTLSKeyFilePath:    s.ServiceHTTPTLSKeyFilePath,
	}
	s.Nodes[clusterNodeIdx].ServiceRaftable = RaftableServiceMaker(clusterNodeIdx, strings.Count(s.InitialMembers, ",")+1, serviceDataPermDir, serviceDataTempDir, s.Nodes[clusterNodeIdx])
	if err = s.Nodes[clusterNodeIdx].Init(); err != nil {
		return nil, err
	}

	// play the inner service before enabling access through outer layers
	if err = s.Nodes[clusterNodeIdx].ServiceRaftable.Play(); err != nil {
		return nil, err
	}
	s.Nodes[clusterNodeIdx].Play()

	// Init and play
	return s.Nodes[clusterNodeIdx], nil
}

// PlayGremlins starts gremlins for chaos testing
func (s *ServiceRaftCluster) PlayGremlins() error {
	core.Log.Warn("GREMLINS : enabled")
	// Wait 10 secs, then start gremlins
	time.Sleep(20 * time.Second)

	gc := &GremlinCage{}
	gc.Init()

	var clusterNodeIdx int
	gc.Gremlins = []Gremlin{
		&CMDKillGremlin{
			KillIt: func(g *CMDKillGremlin) error {
				if s.ForkChildren {
					return g.KillOne(s.ForkCmds[clusterNodeIdx])
				}
				return s.Nodes[clusterNodeIdx].Stop(nil)
			},
			StartIt: func(g *CMDKillGremlin) error {
				if s.ForkChildren {
					args := s.CLIArgsGet(clusterNodeIdx)
					core.Log.Warnf("Forking with %v", args)
					cmd, err := s.RaftableServiceLauncher(clusterNodeIdx, args)
					if err != nil {
						return err
					}
					s.ForkCmds[clusterNodeIdx] = cmd
				} else {
					serviceHTTP, err := s.PlayOne(clusterNodeIdx, s.RaftableServiceMaker)
					if err != nil {
						return err
					}
					s.Nodes[clusterNodeIdx] = serviceHTTP
				}

				clusterNodeIdx = core.RandInt(0, s.NodeCount)
				return nil
			},
		},
		&EthDelay200msGremlin{},
		&EthLose10PctGremlin{},
		&EthDup01PctGremlin{},
		&EthLimit1MbPctGremlin{},
	}

	core.Log.Warn("GREMLINS : starting")
	gc.Play()
	return nil
}
