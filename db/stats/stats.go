package stats

import (
	"context"
	"log"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// Client is an InfluxDB client
type Client struct {
	Client api.WriteAPI

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
	if c == nil {
		return
	}

	name := reflect.ValueOf(ev).Elem().Type().Name()

	c.mu.Lock()
	c.m[name]++
	c.mu.Unlock()
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

	c.mu.Lock()
	im := make(map[string]interface{}, len(c.m))
	for k, v := range c.m {
		im[k] = v
		c.m[k] = 0
	}
	c.mu.Unlock()

	p := influxdb2.NewPoint("events", nil, im, time.Now())
	c.Client.WritePoint(p)
}
