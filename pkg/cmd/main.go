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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/rkosegi/universal-exporter/pkg/internal/server"
	"github.com/rkosegi/universal-exporter/pkg/internal/services"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-toolkit/fluent"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	pv "github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
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
)

func loadConfig(cfgFile string) (*types.Config, error) {
	return fluent.NewConfigHelper[types.Config]().
		Add(server.DefaultConfig()).
		Load(cfgFile).Result(), nil
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
	logger.Info("Starting exporter", "name", name, "version", pv.Info(), "config", *cfgFile)
	logger.Info("Build context", "build_context", pv.BuildContext())

	config, err := loadConfig(*cfgFile)
	if err != nil {
		panic(err)
	}

	if len(config.Targets) == 0 {
		logger.Info("No targets defined")
		os.Exit(1)
	}

	if len(config.Metrics) == 0 {
		logger.Info("No metrics defined")
		os.Exit(1)
	}

	logger.Info("Starting exporter", "targets", len(config.Targets), "metrics", len(config.Metrics))

	r := prometheus.NewRegistry()

	ms := services.NewMetricService(config.Metrics, logger)
	if err = ms.Start(); err != nil {
		logger.Error("Couldn't initialize metric service", "err", err)
		os.Exit(1)
	}

	hcs := services.NewHttpClient(*config.HttpClient, logger, r)
	if err = hcs.Start(); err != nil {
		logger.Error("Couldn't initialize HTTP client service", "err", err)
		os.Exit(1)
	}

	if err = r.Register(server.NewExporter(config, logger, hcs, ms)); err != nil {
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
		if *c.Version {
			r.MustRegister(version.NewCollector(types.PromNamespace))
		}
	}

	landingPageHandler, err := web.NewLandingPage(web.LandingConfig{
		Name:        strings.ReplaceAll(name, "_", " "),
		Description: "Universal exporter for Prometheus",
		Version:     pv.Info(),
		Links: []web.LandingLinks{
			{
				Address: *config.Server.MetricsPath,
				Text:    "Metrics",
			},
			{
				Address: *config.Server.HealthEndpoint,
				Text:    "Health",
			},
		},
	})
	if err != nil {
		logger.Error("Couldn't create landing page", "err", err)
		os.Exit(1)
	}

	http.Handle("/", landingPageHandler)
	http.Handle(*config.Server.MetricsPath, metricHandler)
	http.Handle(*config.Server.HealthEndpoint, healthHandler())

	srv := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err = web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		logger.Error("Error starting server", "err", err)
		os.Exit(1)
	}
}
