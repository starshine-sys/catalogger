package stats

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// Client is an InfluxDB client
type Client struct {
	Client api.WriteAPI

	queriesMu sync.Mutex
	queries   uint32

	cmdsMu sync.Mutex
	cmds   uint32

	m  map[string]uint32
	mu sync.Mutex

	reqs   map[string]uint32
	reqsMu sync.Mutex

	httpMode bool
	httpReqs *uint32

	Counts func() (guilds, channels, roles, messages int64, timeTaken time.Duration)
}

// New creates a new client
func New(url, token, organization, database string) *Client {
	c := &Client{
		m:    make(map[string]uint32),
		reqs: make(map[string]uint32),
		Counts: func() (int64, int64, int64, int64, time.Duration) {
			return 0, 0, 0, 0, 0
		},
		httpReqs: new(uint32),
	}

	c.Client = influxdb2.NewClientWithOptions(url, token,
		influxdb2.DefaultOptions().SetBatchSize(20)).WriteAPI(organization, database)

	go c.submit()

	return c
}

// EventHandler handles Arikawa events
func (c *Client) EventHandler(ev interface{}) {
	c.RegisterEvent(reflect.ValueOf(ev).Elem().Type().Name())
}

// RegisterEvent registers an event name. Separate from EventHandler to allow us to log our own, custom events (currently: nick update + kick)
func (c *Client) RegisterEvent(name string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.m[name]++
	c.mu.Unlock()
}

func (c *Client) IncRequests(method, path string, status int) {
	if c == nil {
		return
	}

	c.reqsMu.Lock()
	c.reqs[EndpointMetricsName(method, path)]++
	c.reqsMu.Unlock()
}

// IncQuery increments the query count by one
func (c *Client) IncQuery() {
	if c == nil {
		return
	}

	c.queriesMu.Lock()
	c.queries++
	c.queriesMu.Unlock()
}

// IncCommand increments the command count by one
func (c *Client) IncCommand() {
	if c == nil {
		return
	}

	c.cmdsMu.Lock()
	c.cmds++
	c.cmdsMu.Unlock()
}

func (c *Client) IncHTTPRequest(req string) {
	if c == nil {
		return
	}

	atomic.AddUint32(c.httpReqs, 1)
}

// True: only send http request counts
func (c *Client) SetMode(http bool) {
	if c == nil {
		return
	}

	c.httpMode = http
}

func (c *Client) submit() {
	if c == nil {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	defer stop()

	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-ticker.C:
			// submit metrics
			go c.submitInner()
		case <-ctx.Done():
			// break if we're shutting down
			ticker.Stop()
			c.Client.Flush()
			return
		}
	}
}

func (c *Client) submitInner() {
	if c == nil {
		return
	}

	log.Println("Submitting metrics to InfluxDB")

	if c.httpMode {
		count := atomic.LoadUint32(c.httpReqs)
		atomic.StoreUint32(c.httpReqs, 0)

		c.Client.WritePoint(influxdb2.NewPoint("http", nil, map[string]interface{}{"http_requests": count}, time.Now()))
		return
	}

	var cmds, queries, totalEvents uint32

	c.queriesMu.Lock()
	queries = c.queries
	c.queries = 0
	c.queriesMu.Unlock()

	c.cmdsMu.Lock()
	cmds = c.cmds
	c.cmds = 0
	c.cmdsMu.Unlock()

	c.mu.Lock()
	im := make(map[string]interface{}, len(c.m))
	for k, v := range c.m {
		totalEvents += v
		im[k] = v
		c.m[k] = 0
	}
	c.mu.Unlock()

	p := influxdb2.NewPoint("events", nil, im, time.Now())
	c.Client.WritePoint(p)

	// requests
	c.reqsMu.Lock()
	rm := make(map[string]interface{}, len(c.reqs))
	for k, v := range c.reqs {
		rm[k] = v
		c.reqs[k] = 0
	}
	c.reqsMu.Unlock()

	c.Client.WritePoint(
		influxdb2.NewPoint("requests", nil, rm, time.Now()),
	)

	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	data := map[string]interface{}{
		"queries":     queries,
		"events":      totalEvents,
		"commands":    cmds,
		"alloc":       stats.Alloc,
		"sys":         stats.Sys,
		"total_alloc": stats.TotalAlloc,
		"goroutines":  runtime.NumGoroutine(),
	}

	sysMem, err := mem.VirtualMemory()
	if err != nil {
		log.Println("Error getting system memory:", err)
	} else {
		data["total_sys"] = sysMem.Used
		data["total_sys_percent"] = sysMem.UsedPercent
	}

	cpuData, err := cpu.Percent(time.Minute, true)
	if err != nil {
		log.Println("Error getting cpu info:", err)
	} else {
		for i, d := range cpuData {
			data[fmt.Sprintf("cpu_%d", i)] = d
		}
	}

	guilds, channels, roles, messages, taken := c.Counts()
	// if the first one isn't 0, we actually got data
	if guilds != 0 {
		data["guilds"] = guilds
		data["channels"] = channels
		data["roles"] = roles
		data["messages"] = messages
		data["query_time"] = int64(taken)
	}

	p = influxdb2.NewPoint("statistics", nil, data, time.Now())
	c.Client.WritePoint(p)
}
