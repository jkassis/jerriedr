package kittie

import (
	"bytes"
	context "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	math "math"
	"net/http"
	"strings"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/lni/dragonboat/v3"
	"github.com/lni/dragonboat/v3/client"
	"github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/raftio"
	"github.com/lni/dragonboat/v3/statemachine"
	"github.com/sirupsen/logrus"
)

// ServiceRaftDBRSM provides service transport over raft
type ServiceRaftDBRSM struct {
	Context               context.Context
	NoOpSession           *client.Session
	NodeHost              *dragonboat.NodeHost
	Playing               bool
	RaftAdvertiseHostport string
	RaftBindHostport      string
	RaftInitialMembers    []string
	RaftJoinHostport      string
	RaftLocalID           string
	RaftLogger            *logrus.Logger
	RaftNodeHostDir       string
	RaftRTTMillisecond    uint64
	RaftRTTsPerElection   int
	RaftRTTsPerHeartbeat  int
	RaftServerResTimeout  time.Duration
	RaftSnapshotDir       string
	RaftSnapshotThreshold int
	RaftSnapshotsToRetain int
	RaftWALDir            string
	RaftableService       *RaftableService
	isJoinedToCluster     bool
	routes                map[string]Handler
}

// ServerResTimeout returns the timeout for raft proposals
func (s *ServiceRaftDBRSM) ServerResTimeout() time.Duration {
	return s.RaftServerResTimeout
}

// ClusterIsJoined returns if the cluster has been joined or not
func (s *ServiceRaftDBRSM) ClusterIsJoined() bool {
	return s.isJoinedToCluster
}

// Routes returns routes for this raft service
func (s *ServiceRaftDBRSM) Routes() map[string]Handler {
	return s.routes
}

// Init sets up a new serviceRaft
func (s *ServiceRaftDBRSM) Init() error {
	s.routes = map[string]Handler{
		"raft/leader/join":  s.LeaderJoinHandler,
		"raft/leader/read":  s.ReadHandler,
		"raft/leader/write": s.WriteHandler,
		// not used...
		// "raft/local/write":  s.WriteHandler,
		// "raft/local/read":   s.ReadHandler,
		// "raft/local/join":   s.LeaderJoinHandler,
	}

	return nil
}

// Play bootstraps the cluster or joins one
func (s *ServiceRaftDBRSM) Play() (err error) {
	if s.Playing {
		return nil
	}
	s.Playing = true

	// Dragonboat seems to do better with 0.0.0.0
	if strings.HasPrefix(s.RaftBindHostport, ":") {
		s.RaftBindHostport = "0.0.0.0" + s.RaftBindHostport
	}

	// Bring up the NodeHost
	nodeHostConfig := config.NodeHostConfig{
		DeploymentID:  10,
		EnableMetrics: true,
		ListenAddress: s.RaftBindHostport,
		// LogDBFactory:      pebble.NewLogDB,
		NodeHostDir:       s.RaftNodeHostDir,
		RTTMillisecond:    s.RaftRTTMillisecond,
		RaftAddress:       s.RaftAdvertiseHostport,
		RaftEventListener: s,
		WALDir:            s.RaftWALDir,
	}
	if nodeHostConfigJSON, err := json.Marshal(map[string]interface{}{
		"EnableMetrics":  nodeHostConfig.EnableMetrics,
		"ListenAddress":  nodeHostConfig.ListenAddress,
		"NodeHostDir":    nodeHostConfig.NodeHostDir,
		"RTTMillisecond": nodeHostConfig.RTTMillisecond,
		"RaftAddress":    nodeHostConfig.RaftAddress,
		"WALDir":         nodeHostConfig.WALDir,
	}); err == nil {
		core.Log.Warnf("ServiceRaftDB: Play: Starting with NodeHostConfig: %s", string(nodeHostConfigJSON))
	} else {
		return err
	}
	nodeHost, err := dragonboat.NewNodeHost(nodeHostConfig)
	if err != nil {
		core.Log.Errorf("ServiceRaftDB: Play: dragonboat.NewNodeHost: %v", err)
		time.Sleep(320 * time.Second)
		return err
	}
	s.NodeHost = nodeHost

	// Bring up the RaftNode
	nodeConfig := config.Config{
		CheckQuorum:            false,
		ClusterID:              DefaultClusterID,
		CompactionOverhead:     uint64(s.RaftSnapshotThreshold),
		DisableAutoCompactions: false,
		ElectionRTT:            20,
		HeartbeatRTT:           2,
		NodeID:                 core.Hash(s.RaftLocalID),
		SnapshotEntries:        uint64(s.RaftSnapshotThreshold),
	}
	if nodeConfigJSON, err := json.Marshal(nodeConfig); err == nil {
		core.Log.Warnf("ServiceRaftDB: Play: Starting with NodeID: %s", s.RaftLocalID)
		core.Log.Warnf("ServiceRaftDB: Play: Starting with NodeConfig: %s", string(nodeConfigJSON))
	} else {
		return err
	}

	// StateMachineFactory function
	stateMachineFactory := func(clusterID uint64, nodeID uint64) statemachine.IStateMachine {
		s.NoOpSession = nodeHost.GetNoOPSession(clusterID)
		return s
	}

	// Try to Start the Cluster without joining and then try to start it with a join
	bootstrapOrJoin := func() error {
		core.Log.Warn("ServiceRaftDB: Play: Cluster state not found.")
		start := time.Now()

		// Parse initial members
		core.Log.Warnf("ServiceRaftDB: Play: Initial members are '%s'", s.RaftInitialMembers)
		if len(s.RaftInitialMembers) == 0 {
			return fmt.Errorf("ServiceRaftDB: no initial members posted")
		}

		initialMembers := make(map[uint64]string)
		for _, initialMemberPair := range s.RaftInitialMembers {
			initialMemberParts := strings.Split(initialMemberPair, "|")
			initialMembers[core.Hash(initialMemberParts[0])] = initialMemberParts[1]
		}

		// Should we init the cluster?
		initCluster := false
		for k, v := range initialMembers {
			if k == nodeConfig.NodeID && v == nodeHostConfig.RaftAddress {
				initCluster = true
				break
			}
		}
		if initCluster {
			core.Log.Warn("ServiceRaftDB: Play: Creating new cluster with initial members")
			if err := nodeHost.StartCluster(initialMembers, false, stateMachineFactory, nodeConfig); err != nil {
				core.Log.Fatal(fmt.Errorf("failed to add cluster: %w", err).Error())
			}
			core.Log.Warnf("ServiceRaftDB: Play: Cluster init complete in %s.", time.Since(start).String())
		} else {
			core.Log.Warn("ServiceRaftDB: Play: Joining an existing cluster.")
			initialMembers := make(map[uint64]string)
			if err := nodeHost.StartCluster(initialMembers, true, stateMachineFactory, nodeConfig); err != nil {
				core.Log.Fatal(fmt.Errorf("failed to add cluster: %w", err).Error())
			}
			core.Log.Warn("ServiceRaftDB: Play: started cluster node with no initial members and waiting in join mode")

			// This sends our raft address to the raft join address... we should be added after this.
			err := s.JoinRequest(s.RaftJoinHostport, s.NodeHost.RaftAddress(), nodeConfig.NodeID, nodeConfig.ClusterID, false)
			if err != nil {
				core.Log.Fatal(err)
			}

			// Now wait for a leadership election
			core.Log.Warnf("ServiceRaftDB: Play: Cluster init complete in %s.", time.Since(start).String())
		}

		return nil
	}

	defer func() {
		err := recover()
		if err != nil {
			errstring, ok := err.(error)
			if !ok || errstring.Error() != "cluster not bootstrapped" {
				panic(err)
			}

			if err := bootstrapOrJoin(); err != nil {
				panic(err)
			}
			s.isJoinedToCluster = true
		}
	}()

	// Start an existing cluster (initialMembers == empty && join == none)
	initialMembers := make(map[uint64]string)
	if err := nodeHost.StartCluster(initialMembers, false, stateMachineFactory, nodeConfig); err != nil {
		return err
	}
	core.Log.Warn("ServiceRaftDB: Play: started cluster node from existing state")
	s.isJoinedToCluster = true

	// // Update my host / IP address if necessary...
	// core.Log.Warn("ServiceRaftDB: Play: GetClusterMembership")
	// var membership *dragonboat.Membership
	// ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	// defer cancel()
	// if membership, err = s.NodeHost.GetClusterMembership(ctx, nodeConfig.ClusterID); err != nil {
	// 	return err
	// }
	// raftAddress := membership.Nodes[nodeConfig.NodeID]
	// if raftAddress == "" || raftAddress != nodeHostConfig.RaftAddress {
	// 	core.Log.Warnf("ServiceRaftDB: Play: My RaftAddress has changed from %s to %s", raftAddress, nodeHostConfig.RaftAddress)
	// }

	// core.Log.Warn("ServiceRaftDB: Play: started cluster node from existing state")
	// s.clusterIsJoined = true

	return nil
}

// LeaderUpdated handles raft leadership changes
func (s *ServiceRaftDBRSM) LeaderUpdated(info raftio.LeaderInfo) {
	defer core.SentryRecover("ServiceRaft.Play.SetLeader")
	s.RaftableService.SetLeader(info.LeaderID == info.NodeID)
}

// Stop stops the service
func (s *ServiceRaftDBRSM) Stop() error {
	if !s.Playing {
		return nil
	}

	s.NodeHost.Stop()
	s.Playing = false
	return nil
}

// Close stops the FSM
func (s *ServiceRaftDBRSM) Close() error {
	return s.RaftableService.Stop()
}

// JoinRequest sends a request to join a cluster (generally through the k8s service address / ingress)
// If this is the first node to join the service, the request will timeout as liveness hasn't allowed
// this to join the LB yet. In that case, bootstrap the cluster.
func (s *ServiceRaftDBRSM) JoinRequest(joinHostport, joinPeerHostport string, nodeID uint64, clusterID uint64, autoJoin bool) error {
	b, err := json.Marshal(JoinRequestData{Hostport: joinPeerHostport, NodeID: nodeID, ClusterID: clusterID})
	if err != nil {
		return err
	}
	joinURL := fmt.Sprintf("http://%s/raft/leader/join", joinHostport)
	if autoJoin {
		core.Log.Warnf("ServiceRaftDB: JoinRequest: Sending a join request to '%s' to join as '%s' with hostport '%s'", s.RaftJoinHostport, s.RaftLocalID, s.RaftAdvertiseHostport)
		resp, err := http.Post(joinURL, "application-type/json", bytes.NewReader(b))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read from body: %v", err)
		}
		core.Log.Warnf("ServiceRaftDB: JoinRequest: got this response: %s", string(body))
		if string(body) != string(core.ResponseOK) {
			return fmt.Errorf("ServiceRaftDB: JoinRequest: %s", body)
		}
	} else {
		core.Log.Warnf("ServiceRaftDB: JoinRequest: To complete the join... post '%s' to '%s' ", string(b), joinURL)
	}
	return nil

	// membership := &dragonboat.Membership{}
	// if err := json.Unmarshal(body, membership); err != nil {
	// 	return nil, err
	// }
	// return membership, nil
}

// LeaderJoinHandler handles a request to join a cluster
// The node must be ready to respond to Raft communications at that address.
func (s *ServiceRaftDBRSM) LeaderJoinHandler(ctx context.Context, req []byte) (res []byte, err error) {
	core.Log.Warnf("ServiceRaftDB: JoinHandler: got request: %s", string(req))

	jr := &JoinRequestData{}
	if err = json.Unmarshal(req, jr); err != nil {
		return nil, fmt.Errorf("could not decode request: %w", err)
	}

	core.Log.Warn("ServiceRaftDB: JoinHandler: RequestAddNode")
	if err = s.NodeHost.SyncRequestAddNode(ctx, jr.ClusterID, jr.NodeID, jr.Hostport, 0); err != nil {
		return nil, err
	}

	// core.Log.Warn("ServiceRaftDB: JoinHandler: GetClusterMembership")
	// var membership *dragonboat.Membership
	// if membership, err = s.NodeHost.GetClusterMembership(ctx, jr.ClusterID); err != nil {
	// 	return nil, err
	// }
	// var resBuffer []byte
	// if resBuffer, err = json.Marshal(membership); err != nil {
	// 	return nil, err
	// }
	core.Log.Warn("ServiceRaftDB: JoinHandler: Send Response: OK")
	return core.ResponseOK, nil
}

// WriteHandler put a request on the raft (only leaders should do this)
func (s *ServiceRaftDBRSM) WriteHandler(ctx context.Context, req []byte) ([]byte, error) {
	start := time.Now()
	defer raftPutTimeMetrc.Observe(float64(time.Since(start)))

	// Get the future and return errors or results
	res, err := s.NodeHost.SyncPropose(ctx, s.NoOpSession, req)
	if err != nil {
		return nil, fmt.Errorf("ServiceRaftDB: WriteHandler: %s", err.Error())
	}
	if res.Value == smResultSuccess {
		return res.Data, nil
	}
	return nil, errors.New(string(res.Data))

	// requestState, err := s.NodeHost.Propose(s.NoOpSession, req, 5*time.Second)
	// if err != nil {
	// 	return nil, err
	// }
	// requestResult := <-requestState.CompletedC
	// res := requestResult.GetResult()

	// if res.Data == nil || len(res.Data) == 0 {
	// 	return nil, fmt.Errorf("got an empty response from the raft for req: %s", string(req))
	// }
}

// ReadHandler get a request from the raft (only leaders should do this)
func (s *ServiceRaftDBRSM) ReadHandler(ctx context.Context, req []byte) ([]byte, error) {
	start := time.Now()
	defer raftGetTimeMetrc.Observe(float64(time.Since(start)))

	// Get the future and return errors or results
	res, err := s.NodeHost.SyncRead(ctx, s.NoOpSession.ClusterID, req)
	if err != nil {
		return nil, fmt.Errorf("ServiceRaftDB: SyncRead: %w", err)
	}
	var result statemachine.Result
	var ok bool
	if result, ok = res.(statemachine.Result); !ok {
		return nil, fmt.Errorf("ServiceRaftDB: ReadHandler: Expected result to be a statemachine.Result: %s", string(req))
	}

	if result.Value == smResultError {
		return nil, errors.New(string(result.Data))
	}

	return result.Data, err

	// requestState, err := s.NodeHost.Propose(s.NoOpSession, req, 5*time.Second)
	// if err != nil {
	// 	return nil, err
	// }
	// requestResult := <-requestState.CompletedC
	// res := requestResult.GetResult()

	// if res.Data == nil || len(res.Data) == 0 {
	// 	return nil, fmt.Errorf("got an empty response from the raft for req: %s", string(req))
	// }
}

// var updateCount int

// Update applys a change to the FSM
func (s *ServiceRaftDBRSM) Update(req []byte) (statemachine.Result, error) {
	start := time.Now()
	defer servicePutTimeMetrc.Observe(float64(time.Since(start)))
	servicePutReqSizeMetrc.Observe(float64(len(req)))

	// Cancel if this takes more than RaftServerResTimeout
	// ctx should have timeout already, so this is just a safety
	ctx, ctxCancelFn := context.WithTimeout(context.Background(), s.RaftServerResTimeout)
	defer ctxCancelFn()

	// Do the write
	handlerRes, handlerErr, writeErr := s.RaftableService.Write(ctx, math.MaxUint64, req, s.RaftableService.Handler)

	// Record the response size.
	servicePutResSizeMetrc.Observe(float64(len(handlerRes)))

	// Pass handler error through as entry.Result.Data
	if handlerErr != nil {
		return statemachine.Result{Value: smResultError, Data: []byte(handlerErr.Error())}, writeErr
	}
	return statemachine.Result{Value: smResultSuccess, Data: handlerRes}, writeErr
}

// Lookup requests data from the FSM, no changes
func (s *ServiceRaftDBRSM) Lookup(input interface{}) (interface{}, error) {
	start := time.Now()
	defer serviceGetTimeMetrc.Observe(float64(time.Since(start)))

	// validate input
	var ok bool
	var req []byte
	if req, ok = input.([]byte); !ok {
		return nil, fmt.Errorf("ServiceRaftDB: Lookup: input must be a []byte")
	}

	serviceGetReqSizeMetrc.Observe(float64(len(req)))
	ctx, ctxCancelFn := context.WithTimeout(context.Background(), s.RaftServerResTimeout)
	defer ctxCancelFn()
	res, err := s.RaftableService.Read(ctx, req, s.RaftableService.Handler)
	serviceGetResSizeMetrc.Observe(float64(len(res)))

	if err == nil {
		return statemachine.Result{Value: smResultSuccess, Data: res}, nil
	}
	return statemachine.Result{Value: smResultError, Data: []byte(err.Error())}, nil
}

// RecoverFromSnapshot loads raft state from a snapshot
func (s *ServiceRaftDBRSM) RecoverFromSnapshot(snapshotStream io.Reader, snapshotFiles []statemachine.SnapshotFile, abort <-chan struct{}) error {
	snapshot, err := s.RaftableService.SnapshotMake()
	if err != nil {
		return err
	}

	return snapshot.Read(snapshotStream)
}

// SaveSnapshot returns a snapshot of the stateful service managed by the raft.
func (s *ServiceRaftDBRSM) SaveSnapshot(snapshotStream io.Writer, snapshotFiles statemachine.ISnapshotFileCollection, abort <-chan struct{}) error {
	snapshot, err := s.RaftableService.SnapshotMake()
	if err != nil {
		return err
	}

	if err := snapshot.Snap(); err != nil {
		return err
	}
	return snapshot.Write(snapshotStream)
}
