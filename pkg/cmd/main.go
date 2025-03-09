/*
Copyright 2025 Richard Kosegi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rkosegi/universal-exporter/pkg/internal"
	"github.com/rkosegi/universal-exporter/pkg/types"

	"github.com/prometheus/client_golang/prometheus/collectors/version"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	pv "github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"gopkg.in/yaml.v3"
)

const (
	name = "universal_exporter"
)

var (
	cfgFile = kingpin.Flag(
		"config-file",
		"Path to config file.",
	).Default("config.yaml").String()
	toolkitFlags = webflag.AddFlags(kingpin.CommandLine, ":9113")

	telemetryPath = kingpin.Flag(
		"web.telemetry-path",
		"Path under which to expose metrics.",
	).Default("/metrics").String()

	// deprecated ,use config
	disableDefaultMetrics = kingpin.Flag(
		"disable-default-metrics",
		"Exclude default metrics about the exporter itself (promhttp_*, process_*, go_*).",
	).Bool()
)

func loadConfig(cfgFile string) (*types.Config, error) {
	var cfg types.Config
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

func main() {
	promlogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(pv.Print(name))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promslog.New(promlogConfig)
	logger.Info(fmt.Sprintf("Starting %s", name),
		"version", pv.Info(),
		"config", *cfgFile)
	logger.Info("Build context", "build_context", pv.BuildContext())

	config, err := loadConfig(*cfgFile)
	if err != nil {
		panic(err)
	}

	logger.Info(fmt.Sprintf("Got %d targets", len(config.Targets)))

	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector(name))

	if err = r.Register(internal.NewExporter(config, logger)); err != nil {
		logger.Error("Couldn't register "+name, "err", err)
		os.Exit(1)
	}

	metricHandler := promhttp.HandlerFor(
		prometheus.Gatherers{r},
		promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		},
	)

	if config.DefaultExporters != nil {
		c := config.DefaultExporters
		if *c.InstrumentHttpHandler {
			metricHandler = promhttp.InstrumentMetricHandler(
				r, metricHandler,
			)
		}
		if *c.Go {
			r.MustRegister(collectors.NewGoCollector())
		}
		if *c.BuildInfo {
			r.MustRegister(collectors.NewBuildInfoCollector())
		}
		if *c.Process {
			r.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		}
	}

	landingPageHandler, err := web.NewLandingPage(web.LandingConfig{
		Name:        strings.ReplaceAll(name, "_", " "),
		Description: "Universal HTTP exporter for Prometheus",
		Version:     pv.Info(),
		Links: []web.LandingLinks{
			{
				Address: *telemetryPath,
				Text:    "Metrics",
			},
			{
				Address: "/health",
				Text:    "Health",
			},
		},
	})
	if err != nil {
		logger.Error("Couldn't create landing page", "err", err)
		os.Exit(1)
	}

	http.Handle("/", landingPageHandler)
	http.Handle(*telemetryPath, metricHandler)
	http.Handle("/health", healthHandler())

	srv := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err = web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		logger.Error("Error starting server", "err", err)
		os.Exit(1)
	}
}
