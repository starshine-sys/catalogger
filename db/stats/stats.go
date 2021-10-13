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
}

// New creates a new client
func New(url, token, organization, database string) *Client {
	c := &Client{
		m: make(map[string]uint32),
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

	p = influxdb2.NewPoint("statistics", nil, data, time.Now())
	c.Client.WritePoint(p)
}
