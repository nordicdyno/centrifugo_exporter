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
	mutex        sync.Mutex
	client       *gocent.Client
	up           prometheus.Gauge
	gaugeMetrics map[string]*prometheus.GaugeVec
}

type centOpts struct {
	uri     string
	secret  string
	timeout time.Duration
}

// NewExporter returns an initialized Exporter.
func NewExporter(opts centOpts) (*Exporter, error) {
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

	cent := gocent.NewClient(uri, opts.secret, opts.timeout)
	if u.Path != "" {
		cent.Endpoint = uri
	}

	gmm := map[string]*prometheus.GaugeVec{
		"client_bytes_in": newGaugeMetric("client_bytes_in_total",
			"number of bytes coming to client API (bytes sent from clients)", nil),
		"client_bytes_out": newGaugeMetric("client_bytes_out_total",
			"number of bytes coming out of client API (bytes sent to clients)", nil),
		"client_num_connect": newGaugeMetric("client_num_connect",
			"number of connections of client API", nil),
		"client_num_msg_published": newGaugeMetric("client_num_msg_published",
			"number of messages published via client API", nil),
		"client_num_msg_queued": newGaugeMetric("client_num_msg_queued",
			"number of messages put into client queues", nil),
		"client_num_msg_sent": newGaugeMetric("client_num_msg_sent",
			"number of messages actually sent to client", nil),
		"client_num_subscribe": newGaugeMetric("client_num_subscribe",
			"subscribes via client API", nil),
		"node_num_clients": newGaugeMetric("node_num_clients",
			"number of connected authorized clients", nil),
		"node_num_unique_clients": newGaugeMetric("node_num_unique_clients",
			"number of unique clients connected", nil),
		"node_num_channels": newGaugeMetric("node_num_channels",
			"number of active channels", nil),
		"node_num_client_msg_published": newGaugeMetric("node_num_client_msg_published",
			"number of messages published", nil),
		"http_api_num_requests": newGaugeMetric("http_api_num_requests",
			"number of requests to server HTTP API", nil),
	}

	return &Exporter{
		client: cent,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of haproxy successful.",
		}),
		gaugeMetrics: gmm,
	}, nil
}

func newGaugeMetric(metricName string, docString string, constLabels prometheus.Labels) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        metricName,
			Help:        docString,
			ConstLabels: constLabels,
		},
		nil,
	)
}

// Describe describes all the metrics ever exported by the Centrifugo exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.gaugeMetrics {
		m.Describe(ch)
	}
	ch <- e.up.Desc()
}

// Collect fetches the stats from configured Centrifugo location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// TODO: check is necessary
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	for _, m := range e.gaugeMetrics {
		m.Reset()
	}
	e.scrape()

	ch <- e.up
	for _, m := range e.gaugeMetrics {
		m.Collect(ch)
	}
}

func (e *Exporter) scrape() {
	metrics, err := nodeMetrics(e.client)
	if err != nil {
		e.up.Set(0)
		log.Printf("Can't scrape centrifugo: %v", err)
		return
	}

	e.up.Set(1)
	for name, value := range metrics {
		if gauge, ok := e.gaugeMetrics[name]; ok {
			gauge.WithLabelValues().Set(value)
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

	exporter, err := NewExporter(opts)
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

	log.Println("Listening on ", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
