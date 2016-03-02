package cli

import (
	"crypto/tls"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"golang.org/x/net/context"
	"gopkg.in/inconshreveable/log15.v2"
	"sourcegraph.com/sourcegraph/appdash"
	"src.sourcegraph.com/sourcegraph/server/serverctx"
	sgxcli "src.sourcegraph.com/sourcegraph/sgx/cli"
	"src.sourcegraph.com/sourcegraph/util/traceutil"
	"src.sourcegraph.com/sourcegraph/util/traceutil/appdashctx"
)

var flushDurationGauge = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: "appdash",
	Subsystem: "process",
	Name:      "flush_duration_seconds",
	Help:      "Duration of executing the Appdash ChunkedCollector.Flush method.",
})

var flushQueueSizeGauge = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "appdash",
	Subsystem: "process",
	Name:      "flush_queue_size",
	Help:      "Size of the Appdash ChunkedCollector.Flush queue (number of collections to occur).",
})

func init() {
	prometheus.MustRegister(flushDurationGauge)
	prometheus.MustRegister(flushQueueSizeGauge)
}

// initClient is called by the appdash_server.go server init func to
// ensure correct ordering (see that func for more info).
func initClient() {
	sgxcli.PostInit = append(sgxcli.PostInit, func() {
		if _, err := sgxcli.Serve.AddGroup("Appdash client", "Appdash client", &clientFlags); err != nil {
			log.Fatal(err)
		}
	})

	sgxcli.ServeInit = append(sgxcli.ServeInit, func() {
		f, err := clientFlags.configure()
		if err != nil {
			log.Fatal("Error configuring appdash client:", err)
		}
		if f == nil {
			return
		}
		serverctx.Funcs = append(serverctx.Funcs, func(ctx context.Context) (context.Context, error) { return f(ctx), nil })
		sgxcli.ClientContext = append(sgxcli.ClientContext, f)
	})
}

var clientFlags ClientConfig

type ClientConfig struct {
	Disable    bool   `long:"appdash.disable-client" description:"disable appdash client"`
	URL        string `long:"appdash.url" description:"externally accessible URL for Appdash's web UI" default:"http://localhost:7800"`
	RemoteAddr string `long:"appdash.remote-collector-addr" description:"collector addr for appdash client to send to"`
	TLS        bool   `long:"appdash.remote-collector-tls" description:"whether to connect to collector via TLS (if so, remote addr must have hostname, not IP addr, for cert verification)"`
	Debug      bool   `long:"appdash.client-debug"`
}

func (f *ClientConfig) configure() (func(context.Context) context.Context, error) {
	if f.Disable {
		log15.Debug("Appdash client is disabled")
		return nil, nil
	}

	url, err := url.Parse(f.URL)
	if err != nil {
		return nil, err
	}

	if f.TLS && url.Scheme != "https" {
		log15.Crit("Appdash remote collector is using TLS, but the web UI URL is not HTTPS. Fix this with --appdash.url=https://...", "at", f.URL)
	}

	var c appdash.Collector
	var proto string
	if f.TLS {
		host, _, err := net.SplitHostPort(f.RemoteAddr)
		if err != nil {
			return nil, err
		}
		c = appdash.NewTLSRemoteCollector(f.RemoteAddr, &tls.Config{ServerName: host})
		proto = "TLS"
	} else {
		c = appdash.NewRemoteCollector(f.RemoteAddr)
		proto = "TCP (non-TLS)"
	}
	log15.Debug("Recording perf traces using Appdash", "collector", f.RemoteAddr, "proto", proto, "at", url)

	c.(*appdash.RemoteCollector).Debug = f.Debug

	c = &appdash.ChunkedCollector{
		Collector:   c,
		MinInterval: 500 * time.Millisecond,
		OnFlush: func(queueSize int) {
			flushDurationGauge.Inc()
			flushQueueSizeGauge.Set(float64(queueSize))
		},
	}

	traceutil.DefaultCollector = c

	return func(ctx context.Context) context.Context {
		ctx = appdashctx.WithAppdashURL(ctx, url)
		ctx = appdashctx.WithCollector(ctx, c)
		return ctx
	}, nil
}
