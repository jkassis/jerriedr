package kittie

import (
	math "math"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/jkassis/jerrie/core"
)

// GremlinCage is a container for gremlins. Don't let them out at night!
type GremlinCage struct {
	Gremlins     []Gremlin
	SleepTimeMin time.Duration
	SleepTimeMax time.Duration
	RandomSeed   int64
	rand         *rand.Rand
}

// Init does what it says
func (c *GremlinCage) Init() {
	c.Gremlins = make([]Gremlin, 0)
	c.RandomSeed = time.Now().Unix()
	c.SleepTimeMin = 20 * time.Second
	c.SleepTimeMax = 30 * time.Second
	c.rand = rand.New(rand.NewSource(99))
}

// Play starts the gremlin cage
func (c *GremlinCage) Play() error {
	go func() {
		for {
			// Get a random gremlin
			i := int(math.Floor(float64(len(c.Gremlins)) * c.rand.Float64()))
			i = 0
			gremlin := c.Gremlins[i]

			// Play it
			err := gremlin.Do()
			if err != nil {
				core.Log.Error(err.Error())
			}

			// Wait
			SleepTime := time.Duration(int64(c.SleepTimeMin) + int64((float64(c.SleepTimeMax)-float64(c.SleepTimeMin))*c.rand.Float64()))
			time.Sleep(SleepTime)
		}
	}()
	return nil
}

// Gremlin creates mischief
type Gremlin interface {
	Do() error
}

// https://netbeez.net/blog/how-to-use-the-linux-traffic-control/
// https://medium.com/@siddontang/use-chaos-to-test-the-distributed-system-linearizability-4e0e778dfc7d
// https://github.com/pingcap/chaos
// https://pingcap.com/blog/chaos-practice-in-tidb/
// https://medium.com/@siddontang/use-chaos-to-test-the-distributed-system-linearizability-4e0e778dfc7d

// EthDelay200msGremlin delays network traffic by 200ms
// qdisc: modify the scheduler (aka queuing discipline)
// add: add a new rule
// dev eth0: rules will be applied on device eth0
// root: modify the outbound traffic scheduler (aka known as the egress qdisc)
// netem: use the network emulator to emulate a WAN property
// delay: the network property that is modified
//
// 200ms: introduce delay of 200 ms with random 10ms uniform variation with correlation value 25% (since network delays are not completely random)
type EthDelay200msGremlin struct{}

// Do runs the gremlin
func (e *EthDelay200msGremlin) Do() error {
	// Do
	core.Log.Warn("GREMLIN : 200ms network delay")
	cmd := exec.Command("tc", strings.Split("qdisc add dev eth0 root netem delay 200ms 10ms 25%", " ")...)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Sleep
	time.Sleep(10 * time.Second)

	// Undo
	core.Log.Warn("GREMLIN : restoring network")
	cmd = exec.Command("tc", "qdisc", "del", "dev", "eth0", "root")
	cmd.Stdout = os.Stdout
	cmd.Run()
	return nil
}

// EthLose10PctGremlin loses 10% of network traffic
type EthLose10PctGremlin struct{}

// Do runs the gremlin
func (g *EthLose10PctGremlin) Do() error {
	// Do
	core.Log.Warn("GREMLIN : 10% packet loss")
	cmd := exec.Command("tc", strings.Split("qdisc add dev eth0 root netem loss 10%", " ")...)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Sleep
	time.Sleep(10 * time.Second)

	// Undo
	core.Log.Warn("GREMLIN : restoring network")
	cmd = exec.Command("tc", "qdisc", "del", "dev", "eth0", "root")
	cmd.Stdout = os.Stdout
	cmd.Run()
	return nil
}

// EthDup01PctGremlin duplicates 1% of network traffic
type EthDup01PctGremlin struct{}

// Do runs the gremlin
func (g *EthDup01PctGremlin) Do() error {
	// Do
	core.Log.Warn("GREMLIN : 1% traffic duplication")
	cmd := exec.Command("tc", strings.Split("qdisc add dev eth0 root netem duplicate 1%", " ")...)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Sleep
	time.Sleep(10 * time.Second)

	// Undo
	core.Log.Warn("GREMLIN : restoring network")
	cmd = exec.Command("tc", "qdisc", "del", "dev", "eth0", "root")
	cmd.Stdout = os.Stdout
	cmd.Run()
	return nil
}

// EthLimit1MbPctGremlin limites network throughput by 1Mb
// tbf: use the token buffer filter to manipulate traffic rates
// rate: sustained maximum rate
// burst: maximum allowed burst
// latency: packets with higher latency get dropped
type EthLimit1MbPctGremlin struct{}

// Do runs the gremlin
func (g *EthLimit1MbPctGremlin) Do() error {
	// Do
	cmd := exec.Command("tc", strings.Split("qdisc add dev eth0 root tbf rate 1mbit burst 32kbit latency 400ms", " ")...)
	cmd.Stdout = os.Stdout
	cmd.Run()

	// Sleep
	time.Sleep(10 * time.Second)

	// Undo
	cmd = exec.Command("tc", "qdisc", "del", "dev", "eth0", "root")
	cmd.Stdout = os.Stdout
	cmd.Run()
	return nil
}

// CMDKillGremlin kills a command
type CMDKillGremlin struct {
	KillIt  func(g *CMDKillGremlin) error
	StartIt func(g *CMDKillGremlin) error
	isDead  bool
}

// Do runs the gremlin
func (g *CMDKillGremlin) Do() error {
	// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
	// nodeCmds[i].Process.Kill() // Not this way... won't kill children
	if g.isDead {
		//  Node survived!
		core.Log.Error("GREMLIN : node still dead")
	} else {
		// Kill it
		core.Log.Warn("GREMLIN : killing node")
		err := g.KillIt(g)
		if err != nil {
			return nil
		}
		g.isDead = true

		// Sleep
		time.Sleep(10 * time.Second)
		// sleepMin := 5 * time.Now().Second()
		// sleepMax := 10 * time.Now().Second()
		// // SleepTime := sleepMin + int64((float64(sleepMax)-float64(sleepMin))*c.rand.Float64())
	}

	if !g.isDead {
		core.Log.Error("GREMLIN : node survived! will try again later")
		return nil
	}

	// Start it again
	core.Log.Warn("GREMLIN : restoring node")
	err := g.StartIt(g)
	if err != nil {
		core.Log.Error(err)
	} else {
		g.isDead = false
	}
	return nil
}

// KillOne shuts a stack down
func (g *CMDKillGremlin) KillOne(cmd *exec.Cmd) error {
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if err != nil {
		return err
	}
	return nil
}
