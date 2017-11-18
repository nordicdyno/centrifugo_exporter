package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/centrifugal/gocent"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "centrifugo"
)

var VERSION = "unknown"

// Exporter collects Centrifugo stats from the given server and exports them using
// the prometheus metrics package.
type Exporter struct {
	mutex   sync.Mutex
	client  *gocent.Client
	up      prometheus.Gauge
	metrics map[string]metricDesc
}

type centOpts struct {
	uri     string
	secret  string
	timeout time.Duration
}

func newCentClient(opts centOpts) (*gocent.Client, error) {
	uri := opts.uri
	if !strings.Contains(uri, "://") {
		uri = "http://" + uri
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid centrifugo URL: %s", err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("invalid centrifugo URL: %s", uri)
	}

	c := gocent.NewClient(uri, opts.secret, opts.timeout)
	if u.Path != "" {
		c.Endpoint = uri
	}
	return c, nil
}

type metricDesc struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

func newGaugeDesc(name string, desc string) metricDesc {
	return metricDesc{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", name),
			desc,
			nil, nil,
		),
		valueType: prometheus.GaugeValue,
	}
}

func newCounterDesc(name string, desc string) metricDesc {
	return metricDesc{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", name),
			desc,
			nil, nil,
		),
		valueType: prometheus.CounterValue,
	}
}

// NewExporter returns an initialized Exporter.
func NewExporter(centClient *gocent.Client) (*Exporter, error) {
	m := map[string]metricDesc{
		"client_bytes_in": newCounterDesc(
			"client_bytes_in_total",
			"number of bytes coming to client API (bytes sent from clients)"),
		"client_bytes_out": newCounterDesc(
			"client_bytes_out_total",
			"number of bytes coming out of client API (bytes sent to clients)"),
		"client_num_connect": newCounterDesc(
			"client_num_connect",
			"number of connections of client API"),
		"client_num_msg_published": newCounterDesc(
			"client_num_msg_published",
			"number of messages published via client API"),
		"client_num_msg_queued": newCounterDesc(
			"client_num_msg_queued",
			"number of messages put into client queues"),
		"client_num_msg_sent": newCounterDesc(
			"client_num_msg_sent",
			"number of messages actually sent to client"),
		"client_num_subscribe": newCounterDesc(
			"client_num_subscribe",
			"subscribes via client API"),
		"node_num_clients": newGaugeDesc(
			"node_num_clients",
			"number of connected authorized clients"),
		"node_num_unique_clients": newGaugeDesc(
			"node_num_unique_clients",
			"number of unique clients connected"),
		"node_num_channels": newGaugeDesc(
			"node_num_channels",
			"number of active channels"),
		"node_num_client_msg_published": newCounterDesc(
			"node_num_client_msg_published",
			"number of messages published"),
		"http_api_num_requests": newCounterDesc(
			"http_api_num_requests",
			"number of requests to server HTTP API"),
	}

	return &Exporter{
		client: centClient,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of centrifugo successful.",
		}),
		metrics: m,
	}, nil
}

// Describe describes all the metrics ever exported by the Centrifugo exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.metrics {
		ch <- m.desc
		// m.Describe(ch)
	}
	ch <- e.up.Desc()
}

// Collect fetches the stats from configured Centrifugo location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	nodemetrics, scrapeErr := nodeMetrics(e.client)
	e.up.Set(1)
	if scrapeErr != nil {
		e.up.Set(0)
		log.Printf("Can't scrape centrifugo on %v: %v", e.client.Endpoint, scrapeErr)
	}

	ch <- e.up
	if scrapeErr != nil {
		return
	}
	for name, value := range nodemetrics {
		if m, ok := e.metrics[name]; ok {
			// gauge.WithLabelValues().Set(value)
			ch <- prometheus.MustNewConstMetric(m.desc, m.valueType, value)
		}
	}
}

func main() {
	var (
		showVersion   = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9273", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")

		opts = centOpts{}
	)
	flag.StringVar(&opts.uri, "centrifugo.server", "http://localhost:8000", "HTTP API address of a centrifugo server. (prefix with https:// to connect over HTTPS)")
	flag.StringVar(&opts.secret, "centrifugo.secret", "", "centrifugo secret token")
	flag.DurationVar(&opts.timeout, "centrifugo.timeout", 200*time.Millisecond, "Timeout on HTTP requests to centrifugo.")

	flag.Parse()

	if *showVersion {
		fmt.Println("version", VERSION)
		os.Exit(0)
	}

	c, err := newCentClient(opts)
	if err != nil {
		log.Fatalln(err)
	}
	exporter, err := NewExporter(c)
	if err != nil {
		log.Fatalln(err)
	}
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Centrifugo Exporter</title></head>
             <body>
             <h1>Centrifugo Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Println("listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
