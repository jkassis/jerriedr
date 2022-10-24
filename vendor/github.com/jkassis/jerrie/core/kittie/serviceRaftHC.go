package kittie

import (
	"bufio"
	"bytes"
	context "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	raftbadger "github.com/bbva/raft-badger"
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/hashicorp/raft"
	"github.com/jkassis/jerrie/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// ERRJoinNotRequired says it all
var ERRJoinNotRequired = []byte("join not required")

// ServiceRaftHC provides service transport over raft
type ServiceRaftHC struct {
	clusterIsJoined           bool
	Context                   context.Context
	IsLeaderCh                chan bool
	PeerNetRPCClients         map[string]*rpc.Client
	PeerNetRPCClientsSync     sync.Mutex
	PeerServiceNetRPCHostport string
	Playing                   bool
	Raft                      *raft.Raft // The consensus mechanism
	RaftAdvertiseHostport     string
	RaftBindHostport          string
	RaftDBDir                 string
	RaftInitialMembers        []string
	RaftJoinHostport          string
	RaftLocalID               string
	RaftLogger                *logrus.Logger
	RaftServerResTimeout      time.Duration
	RaftSnapshotDir           string
	RaftSnapshotThreshold     int
	RaftSnapshotsToRetain     int
	RaftStore                 *raftbadger.BadgerStore
	RaftTransport             *raft.NetworkTransport
	RaftableService           *RaftableService
	routes                    map[string]Handler

	metrxForwardToLeader *prometheus.CounterVec
}

// ServerResTimeout returns the timeout for raft proposals
func (s *ServiceRaftHC) ServerResTimeout() time.Duration {
	return s.RaftServerResTimeout
}

// ClusterIsJoined returns if the cluster has been joined or not
func (s *ServiceRaftHC) ClusterIsJoined() bool {
	return s.clusterIsJoined
}

// Init sets up a new serviceRaft
func (s *ServiceRaftHC) Init() error {
	s.routes = map[string]Handler{
		"raft/leader/join":  s.JoinHandler,
		"raft/leader/read":  s.ReadHandler,
		"raft/leader/write": s.WriteHandler,
		"raft/local/join":   s.LocalJoinHandler,
		"raft/local/read":   s.LocalReadHandler,
		"raft/local/write":  s.LocalWriteHandler,
	}

	config := raft.DefaultConfig()
	// when we delete a pod in kubernetes and create a new one, it comes back with a different IP.
	// because we remove raft node and add it back in in this case, don't ShutdownOnRemove.
	// we want the raft to continue to take traffic. if this is true, the raft will never take
	// traffic again when it is removed from the cluster
	config.ShutdownOnRemove = false
	// config.ElectionTimeout
	// paranoid set to false explicitly
	// config.StartAsLeader = false
	config.LocalID = raft.ServerID(s.RaftLocalID)
	config.ProtocolVersion = 3
	config.Logger = &logrus.AdapterHCLog{
		MyLogger: s.RaftLogger,
		MyName:   "raft",
	}
	config.SnapshotThreshold = uint64(s.RaftSnapshotThreshold)

	// Forward leadership changes to the service
	s.IsLeaderCh = make(chan bool, 0)
	config.NotifyCh = s.IsLeaderCh
	go func() {
		defer core.SentryRecover("ServiceRaft.Play.SetLeader")
		for {
			isLeader := <-s.IsLeaderCh
			s.RaftableService.SetLeader(isLeader)
		}
	}()

	// Create the snapshot store.
	snapshots, err := raft.NewFileSnapshotStore(s.RaftSnapshotDir, s.RaftSnapshotsToRetain, os.Stderr)
	if err != nil {
		return fmt.Errorf("file snapshot store: '%s'", err)
	}

	// Create the log store and stable store.
	var logStore raft.LogStore
	var stableStore raft.StableStore
	if false {
		logStore = raft.NewInmemStore()
		stableStore = raft.NewInmemStore()
	} else {
		opts := badger.DefaultOptions(s.RaftDBDir)
		opts = opts.WithLogger(s.RaftLogger)
		opts = opts.WithSyncWrites(true)
		opts = opts.WithValueLogLoadingMode(options.FileIO)
		opts = opts.WithTableLoadingMode(options.FileIO)

		raftStore, err := raftbadger.New(raftbadger.Options{
			BadgerOptions: &opts,
			NoSync:        false,
		})
		if err != nil {
			return err
		}
		logStore = raftStore
		stableStore = raftStore
		s.RaftStore = raftStore
	}

	// Setup RaftTransport.
	advertiseAddr, err := net.ResolveTCPAddr("tcp", s.RaftAdvertiseHostport)
	if err != nil {
		return err
	}

	// transport, err := raft.NewTCPTransportWithLogger(serviceRaft.RaftBindHostport, advertiseAddr, 3, 10*time.Second, serviceRaft.RaftLogger)
	transport, err := raft.NewTCPTransport(s.RaftBindHostport, advertiseAddr, 3, 10*time.Second, os.Stdout)
	if err != nil {
		return err
	}
	s.RaftTransport = transport

	// Instantiate the Raft systems
	ra, err := raft.NewRaft(config, raft.FSM(s), logStore, stableStore, snapshots, transport)
	if err != nil {
		return err
	}
	s.Raft = ra

	// Create the map to PeerProxyServiceServers
	s.PeerNetRPCClientsSync.Lock()
	s.PeerNetRPCClients = make(map[string]*rpc.Client)
	s.PeerNetRPCClientsSync.Unlock()

	// counter for redirect rate
	s.metrxForwardToLeader = core.PromRegisterCollector(prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "raft_redirects",
			Help: "calls to callbacks associated with timers",
		}, []string{"attempt"})).(*prometheus.CounterVec)
	return err

	// make a new proxy service server

	// // Make an observer for peer events
	// observationCh := make(chan raft.Observation, 0)
	// observer := raft.NewObserver(observationCh, true, nil)
	// ra.RegisterObserver(observer)

	// go func() {
	// 	for observation := range observationCh {
	// 		core.Log.Trace("got observation" + observation.Raft.LastContact().String())
	// 	}
	// }()
}

// Routes returns routes for this raft service
func (s *ServiceRaftHC) Routes() map[string]Handler {
	return s.routes
}

// Play bootstraps the cluster or joins one
func (s *ServiceRaftHC) Play() error {
	if s.Playing {
		return nil
	}
	s.Playing = true

	core.Log.Warnf("ServiceRaftHC: Play: Getting RAFT configuration")
	future := s.Raft.GetConfiguration()
	if err := future.Error(); err != nil {
		// Couldn't. Problem!
		return fmt.Errorf("ServiceRaft.Play: raft.GetConfiguration failed : %v", err)
	}
	configuration := future.Configuration()

	core.Log.Warnf("ServiceRaftHC: Play: Does the cluster have existing state?")
	if len(configuration.Servers) == 0 {
		core.Log.Warn("ServiceRaftHC: Play: Cluster state not found.")

		// Parse initial members
		if len(s.RaftInitialMembers) == 0 {
			return fmt.Errorf("ServiceRaftHC: Play: no initial members posted")
		}

		initialMembers := []raft.Server{}
		for _, initialMemberPair := range s.RaftInitialMembers {
			initialMemberParts := strings.Split(initialMemberPair, "|")
			initialMembers = append(initialMembers, raft.Server{ID: raft.ServerID(initialMemberParts[0]), Address: raft.ServerAddress(initialMemberParts[1])})
		}

		// Should we init the cluster?
		initCluster := false
		for _, v := range initialMembers {
			if v.ID == raft.ServerID(s.RaftLocalID) && v.Address == raft.ServerAddress(s.RaftAdvertiseHostport) {
				initCluster = true
				break
			}
		}
		if initCluster {
			core.Log.Warn("ServiceRaftHC: Play: Creating new cluster with initial members")
			configuration := raft.Configuration{Servers: initialMembers}
			configFuture := s.Raft.BootstrapCluster(configuration)
			if configFuture.Error() != nil {
				return configFuture.Error()
			}
			// wait for leadership
			core.Log.Warn("ServiceRaftHC: Play: Initialized!")
			s.clusterIsJoined = true
		} else {
			core.Log.Warnf("ServiceRaftHC: Play: No. Sending a join request to '%s' to join as '%s' with hostport '%s'", s.RaftJoinHostport, s.RaftLocalID, s.RaftAdvertiseHostport)
			if err := s.JoinRequest(s.RaftJoinHostport, s.RaftAdvertiseHostport, s.RaftLocalID, false); err != nil {
				return err
			}
			core.Log.Warnf("ServiceRaftHC: Play: Join request sent")

			// Block until we have a state
			var i int
			for i = 0; i < 20; i++ {
				leader := s.Raft.Leader()
				if leader != "" {
					break
				}
				time.Sleep(time.Second)
			}
			if i == 20 {
				return fmt.Errorf("ServiceRaftHC: Play: was not contacted by the leader in %dsecs", i)
			}

			core.Log.Warn("ServiceRaftHC: Play: Initialized!")
			s.clusterIsJoined = true
		}
	} else {
		core.Log.Warnf("ServiceRaftHC: Play: Yes. Does it have this node?")
		for _, server := range configuration.Servers {
			core.Log.Warnf("ServiceRaftHC: Play: The cluster includes server.ID '%s' with hostport '%s'", server.ID, server.Address)
			if server.ID == raft.ServerID(s.RaftLocalID) {
				core.Log.Warnf("ServiceRaftHC: Play: Yes. I joined the cluster in a previous run.")
				if string(server.Address) == s.RaftAdvertiseHostport {
					core.Log.Warnf("ServiceRaftHC: Play: Everything is A OK!")
				} else {
					core.Log.Warnf("ServiceRaftHC: Play: The previous address for my node.ID '%s' was '%s', but my current address is '%s'", server.ID, server.Address, s.RaftAdvertiseHostport)
					core.Log.Warnf("ServiceRaftHC: Play: Sending a join request to '%s' to update my address as '%s' with hostport '%s'", s.RaftJoinHostport, s.RaftLocalID, s.RaftAdvertiseHostport)
					if err := s.JoinRequest(s.RaftJoinHostport, s.RaftAdvertiseHostport, s.RaftLocalID, false); err != nil {
						return err
					}
					core.Log.Warnf("ServiceRaftHC: Play: Join request sent")
					core.Log.Warn("ServiceRaftHC: Play: Initialized!")
				}
				s.clusterIsJoined = true
				return nil
			}
		}
		return fmt.Errorf("ServiceRaftHC: Play: The local cluster configuration was initialized with servers, but did not have an entry for this local cluster ID. This looks like a configuration error or a problem with stale cluster configuration data on disk. Wipe the disk and try again")
	}

	return nil
}

// Stop stops the service
func (s *ServiceRaftHC) Stop() error {
	if !s.Playing {
		return nil
	}

	f := s.Raft.Shutdown()
	err := f.Error()
	if err != nil {
		return err
	}

	if s.RaftStore != nil {
		if err := s.RaftStore.Close(); err != nil {
			return err
		}
	}

	s.Playing = false
	return nil
}

func (s *ServiceRaftHC) proxyServiceServerHandleConn(conn net.Conn) {
	core.Log.Trace("ServiceRaftHC: proxyServiceServerHandleConn")
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer conn.Close()

	// Read from the connection until EOF. Expect a command name as the next input. Call the handler that is registered for this command.
	for {
		log.Print("ServiceRaftHC: Receive command '")
		cmd, err := rw.ReadString('\n')
		switch {
		case err == io.EOF:
			log.Println("ServiceRaftHC: Reached EOF - close this connection.\n   ---")
			return
		case err != nil:
			log.Println("ServiceRaftHC: \nError reading command. Got: '"+cmd+"'\n", err)
			return
		}
	}
}

func (s *ServiceRaftHC) peerNetRPCServiceHostport(raftServiceHostport string) (proxyServiceHostport string, err error) {
	return HostportDelta(raftServiceHostport, 3)
}

// PeerNetRPCClientGet ensures that a connection to the peer's leader proxy server exists
func (s *ServiceRaftHC) PeerNetRPCClientGet(peerHostport string) (*rpc.Client, error) {
	// Already got one?
	s.PeerNetRPCClientsSync.Lock()
	defer s.PeerNetRPCClientsSync.Unlock()
	if s.PeerNetRPCClients[peerHostport] == nil {
		// Nope. Make one.
		peerNetRPCServiceHostport, err := s.peerNetRPCServiceHostport(peerHostport)
		if err != nil {
			return nil, err
		}
		client, err := rpc.Dial("tcp", peerNetRPCServiceHostport)
		if err != nil {
			return nil, err
		}
		s.PeerNetRPCClients[peerHostport] = client
	}

	return s.PeerNetRPCClients[peerHostport], nil
}

// IsLeader returns a bool to indicate if this raft is the leader
func (s *ServiceRaftHC) IsLeader() bool {
	return s.Raft.State() == raft.Leader
}

// JoinHandler handle request to join the raft
func (s *ServiceRaftHC) JoinHandler(ctx context.Context, req []byte) (res []byte, err error) {
	core.Log.Trace("ServiceRaftHC: JoinHandler: got request")
	start := time.Now()
	defer raftJoinTimeMetrc.Observe(float64(time.Since(start)))
	resFromForward, err := s.leaderForwardHandler(ctx, "raft/local/join.HandlePost", req, func() (res []byte, err error) {
		return s.LocalWriteHandler(ctx, req)
	})
	return resFromForward, err
}

// ReadHandler performs a serialized / linearizable read
func (s *ServiceRaftHC) ReadHandler(ctx context.Context, req []byte) (res []byte, err error) {
	core.Log.Trace("ServiceRaftHC: LeaderRaftAPIHandler: got request")
	start := time.Now()
	defer raftGetTimeMetrc.Observe(float64(time.Since(start)))
	resFromForward, err := s.leaderForwardHandler(ctx, "raft/local/read.HandlePost", req, func() (res []byte, err error) {
		return s.LocalWriteHandler(ctx, req)
	})
	return resFromForward, err
}

// WriteHandler forwards the request to the leader or apply to the raft (if the leader)
func (s *ServiceRaftHC) WriteHandler(ctx context.Context, req []byte) (res []byte, err error) {
	core.Log.Trace("ServiceRaftHC: LeaderRaftAPIHandler: got request")
	start := time.Now()
	defer raftPutTimeMetrc.Observe(float64(time.Since(start)))
	resFromForward, err := s.leaderForwardHandler(ctx, "raft/local/write.HandlePost", req, func() (res []byte, err error) {
		return s.LocalWriteHandler(ctx, req)
	})
	return resFromForward, err
}

// ServiceHandler forwards the request to the leader or directly to the service, skipping the raft.
func (s *ServiceRaftHC) ServiceHandler(ctx context.Context, req []byte) (res []byte, err error) {
	return s.leaderForwardHandler(ctx, "service.HandlePost", req, func() (res []byte, err error) {
		return s.RaftableService.Handler(ctx, req)
	})
}

func (s *ServiceRaftHC) leaderForwardHandler(ctx context.Context, route string, req []byte,
	// TODO It's a little odd that this signature doesn't accept a context.
	// But this is a private function and the way this is used in this class
	// is that the handlers are context-bound closures
	handler func() (res []byte, err error)) (res []byte, err error) {

	//try to shortcut
	if s.Raft.State() != raft.Leader {
		return s.leaderForward(ctx, route, req)
	}

	// give it a try on this node
	res, err = handler()
	if err != nil {
		// even though we tried the shortcut, we still might not be the leader
		if err.Error() == "node is not the leader" {
			return s.leaderForward(ctx, route, req)
		}
		return nil, err
	}
	return res, nil
}

// this fn attempts to forward requests to the leader of the raft and retrys a number of times
// TODO This should probably just retry until the context expires.
func (s *ServiceRaftHC) leaderForward(ctx context.Context, route string, req []byte) (res []byte, err error) {
	rpcReq := &ServiceNetRPCPostRequest{Data: req}
	rpcRes := &ServiceNetRPCPostResponse{}

	var attempt int
	tryOne := func() error {
		// TODO
		s.metrxForwardToLeader.WithLabelValues(strconv.Itoa(attempt)).Inc()

		// No... Get leaderHostport and validate that it is not the local address
		leaderHostport := string(s.Raft.Leader())
		if leaderHostport == "" || leaderHostport == s.RaftAdvertiseHostport {
			core.Log.Warn("ServiceRaftHC: leaderForward: leader unknown")
			return errors.New("leader unknown")
		}

		if strings.HasSuffix(s.RaftAdvertiseHostport, leaderHostport) {
			// this should *never* happen,  unless misconfigured.
			core.Log.Error("ServiceRaftHC: leaderForward: local node is the leader. refusing to forward to self. misconfigured advertise address?")
			return errors.New("leader unknown")
		}

		// Get a client to the leader's NetRPCService
		core.Log.Tracef("ServiceRaftHC: leaderForward: forwarding to %s", leaderHostport)
		client, err := s.PeerNetRPCClientGet(leaderHostport)
		if err != nil {
			return err
		}

		// Call
		return client.Call(route, rpcReq, rpcRes)
	}

	for attempt = 1; attempt < 9; attempt++ {
		err := tryOne()
		if err == nil {
			break
		}
		if err.Error() == "node is not the leader" {
			core.Log.Warnf("ServiceRaftHC: leaderForward: %s", err.Error())
			core.Log.Warnf("ServiceRaftHC: leaderForward: sleeping")
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * 50 * time.Millisecond)
			core.Log.Warnf("ServiceRaftHC: leaderForward: attempt %d: retrying", attempt)
		} else {
			return nil, err
		}
	}

	if attempt == 9 {
		return nil, errors.New("ServiceRaftHC: leaderForward: too many attempts")
	}

	// Write output if any
	return rpcRes.Data, nil
}

// LocalReadHandler put a request on the raft (only leaders should do this)
func (s *ServiceRaftHC) LocalReadHandler(ctx context.Context, req []byte) ([]byte, error) {
	if s.Raft.State() != raft.Leader {
		return nil, raft.ErrNotLeader
	}

	if core.Log.IsLevelEnabled(logrus.DebugLevel) {
		leaderHostport := string(s.Raft.Leader())
		core.Log.Tracef("ServiceRaftHC: LocalReadHandler: %s is running route with leader %s", s.RaftBindHostport, leaderHostport)
	}

	// We are the leader... apply to the raft
	// Get the future and return errors or results
	deadline, ok := ctx.Deadline()
	if !ok {
		core.Log.Fatal("ServiceRaftHC: LocalReadHandler: No deadline set")
	}
	f := s.Raft.Barrier(time.Until(deadline))
	if f.Error() != nil {
		return nil, f.Error()
	}

	return s.RaftableService.Handler(ctx, req)
}

// LocalWriteHandler put a request on the raft (only leaders should do this)
func (s *ServiceRaftHC) LocalWriteHandler(ctx context.Context, req []byte) ([]byte, error) {
	if s.Raft.State() != raft.Leader {
		return nil, raft.ErrNotLeader
	}

	if core.Log.IsLevelEnabled(logrus.DebugLevel) {
		leaderHostport := string(s.Raft.Leader())
		core.Log.Tracef("ServiceRaftHC: LocalApplyHandler: %s is running route with leader %s", s.RaftBindHostport, leaderHostport)
	}

	// We are the leader... apply to the raft
	// Get the future and return errors or results
	deadline, ok := ctx.Deadline()
	if !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.RaftServerResTimeout)
		defer cancel()
	}
	f := s.Raft.Apply(req, time.Until(deadline))
	if f.Error() != nil {
		return nil, f.Error()
	}
	res := f.Response()

	if res == nil {
		return nil, errors.New("got an empty response from the raft")
	}
	switch res.(type) {
	case []byte:
		return res.([]byte), nil
	case error:
		return nil, res.(error)
	default:
		return nil, errors.New("got unexpected response type from raft")
	}
}

// LocalJoinHandler handles a request to join a cluster
// The node must be ready to respond to Raft communications at that address.
func (s *ServiceRaftHC) LocalJoinHandler(ctx context.Context, req []byte) (res []byte, err error) {
	if s.Raft.State() != raft.Leader {
		return nil, raft.ErrNotLeader
	}

	core.Log.Warn("ServiceRaftHC: LocalJoinHandler: starting")

	jr := &JoinRequestDataHC{}
	if err = json.Unmarshal(req, jr); err != nil {
		return nil, fmt.Errorf("could not decode request: %w", err)
	}

	core.Log.Warn("ServiceRaftHC: LocalJoinHandler: got join request for '", jr.NodeID, "' at '", jr.Hostport, "'")

	configFuture := s.Raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.RaftServerResTimeout)
		defer cancel()
	}

	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(jr.NodeID) && srv.Address == raft.ServerAddress(jr.Hostport) {
			core.Log.Warnf("ServiceRaftHC: LocalJoinHandler: '%s' at '%s' already joined", jr.NodeID, jr.Hostport)
			return ERRJoinNotRequired, nil
		}

		// does another node own this address?
		if srv.Address == raft.ServerAddress(jr.Hostport) {
			err = fmt.Errorf("ServiceRaftHC: LocalJoinHandler: '%s' already owns address '%s'", srv.ID, srv.Address)
			core.Log.Warnf(err.Error())
			return nil, err
		}

		// is this the nodeID with the bad address?
		if srv.ID == raft.ServerID(jr.NodeID) {
			// The docs say that AddVoter will update the node's address, but practice proves that's not true.
			// So we Remove it from the Server before adding it back in with a new IP address.
			core.Log.Warnf("ServiceRaftHC: LocalJoinHandler: '%s' at '%s' conflicts. removing it first.", srv.ID, srv.Address)
			future := s.Raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return nil, fmt.Errorf("ServiceRaftHC: LocalJoinHandler: error removing '%s' at '%s': '%s'", srv.ID, srv.Address, err)
			}
		}
	}

	// All clear... this will add the node or update the node's address. See the GODOCs
	f := s.Raft.AddVoter(raft.ServerID(jr.NodeID), raft.ServerAddress(jr.Hostport), 0, 0)
	if f.Error() != nil {
		core.Log.Warnf("ServiceRaftHC: LocalJoinHandler: failed to add '%s' at '%s': %s", jr.NodeID, jr.Hostport, f.Error().Error())
		return nil, f.Error()
	}
	core.Log.Warnf("ServiceRaftHC: LocalJoinHandler: added '%s' at '%s'. joined successfully", jr.NodeID, jr.Hostport)
	return core.ResponseOK, nil
}

// JoinRequestDataHC data for a join request
type JoinRequestDataHC struct {
	Hostport string
	NodeID   string
}

// JoinRequest sends a request to join a cluster (generally through the k8s service address / ingress)
// If this is the first node to join the service, the request will timeout as liveness hasn't allowed
// this to join the LB yet. In that case, bootstrap the cluster.
func (s *ServiceRaftHC) JoinRequest(joinHostport, joinPeerHostport, nodeID string, autoJoin bool) error {
	b, err := json.Marshal(JoinRequestDataHC{Hostport: joinPeerHostport, NodeID: nodeID})
	if err != nil {
		return err
	}
	joinURL := fmt.Sprintf("http://%s/raft/leader/join", joinHostport)
	if autoJoin {
		tryOne := func() error {
			core.Log.Warnf("ServiceRaftHC: JoinRequest: tryOne: start")
			resp, err := http.Post(joinURL, "application-type/json", bytes.NewReader(b))
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("unable to read from body: %v", err)
			}
			if bytes.Equal(body, core.ResponseOK) {
				core.Log.Warn("ServiceRaftHC: JoinRequest: tryOne: success")
				return nil
			}
			if bytes.Equal(body, ERRJoinNotRequired) {
				core.Log.Warn("ServiceRaftHC: JoinRequest: tryOne: already joined. success.")
				return nil
			}
			err = fmt.Errorf("ServiceRaftHC: JoinRequest: tryOne: %s", string(body))
			core.Log.Warnf(err.Error())
			return err
		}

		// Try 50 times because we may be going through a load balancer to find the leader.
		// With 5 nodes, the chance of missing the leader once is 4/5 (actually 3/5).
		// If we try 50 times... there is a 0.001427247693 percent chance of never hitting the leader
		for i := 0; i < 50; i++ {
			if err := tryOne(); err != nil {
				if err.Error() != raft.ErrNotLeader.Error() && err.Error() != "leader unknown" {
					return err
				}
				core.Log.Warnf("ServiceRaftHC: JoinRequest: sleeping before retry")
				time.Sleep(time.Duration(math.Pow(2, float64(i))) * 50 * time.Millisecond)
				core.Log.Warnf("ServiceRaftHC: JoinRequest: retrying")
			} else {
				break
			}
		}
	} else {
		core.Log.Warnf("ServiceRaftHC: JoinRequest: To complete the join... post '%s' to '%s' ", string(b), joinURL)
	}

	return nil
}

// Bootstrap starts the first node of the first cluster
func (s *ServiceRaftHC) Bootstrap(joinHostport, raftAddr, nodeID string) error {
	// configFuture := serviceRaft.Raft.GetConfiguration()
	// if err := configFuture.Error(); err != nil {
	// 	core.Log.Fatal("failed to get raft configuration: %v", err)
	// 	return err
	// }
	// config := configFuture.Configuration()

	return nil
}

// Apply applies a Raft log entry to the FSM
func (s *ServiceRaftHC) Apply(l *raft.Log) (res interface{}) {
	ctx, ctxCancelFn := context.WithTimeout(context.Background(), s.RaftServerResTimeout)
	defer func() {
		// Always cancel the context
		ctxCancelFn()

		// Catch raft panics. These are unexpected programming problems that need to be fixed.
		// They should be reported generically so that they cannot be "handled" by the client.
		if r := recover(); r != nil {
			crash, _ := strconv.ParseBool(os.Getenv("SERVICE_CRASH_ON_RUNTIME_ERRORS"))
			if crash {
				panic(r)
			}
			switch r.(type) {
			case error:
				res = fmt.Errorf("ServiceRaftHC: Apply: internal error: %w", r.(error))
				core.Log.Error(res)
			default:
				res = fmt.Errorf("ServiceRaftHC: Apply: internal error")
				core.Log.Error(res, r)
			}
		}
	}()

	// Run the handler and return the error if given or the response
	var err error
	start := time.Now()
	res, err = s.RaftableService.Handler(ctx, l.Data)
	res = core.ResponseOK
	duration := time.Since(start)
	servicePutTimeMetrc.Observe(float64(duration))
	if err != nil {
		return err
	}
	return res
}

// Restore sets the stateful service managed by the raft to a previous state.
func (s *ServiceRaftHC) Restore(rc io.ReadCloser) error {
	snapshot, err := s.RaftableService.SnapshotMake()
	if err != nil {
		return err
	}

	return snapshot.Restore(rc)
}

// Snapshot returns a snapshot of the stateful service managed by the raft.
// https://discuss.dgraph.io/t/badger-backup-takes-long-time-to-backup-3g-store/4301/11
func (s *ServiceRaftHC) Snapshot() (raft.FSMSnapshot, error) {
	return s.RaftableService.SnapshotMake()
}
