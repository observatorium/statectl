// Copyright (c) The Observatorium Authors.
// Licensed under the Apache License 2.0.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/observatorium/statectl/pkg/extkingpin"
	"github.com/observatorium/statectl/pkg/project"
	"github.com/observatorium/statectl/pkg/project/loader"
	"github.com/observatorium/statectl/pkg/version"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	logFormatLogfmt = "logfmt"
	logFormatJson   = "json"
)

func setupLogger(logLevel, logFormat string) log.Logger {
	var lvl level.Option
	switch logLevel {
	case "error":
		lvl = level.AllowError()
	case "warn":
		lvl = level.AllowWarn()
	case "info":
		lvl = level.AllowInfo()
	case "debug":
		lvl = level.AllowDebug()
	default:
		panic("unexpected log level")
	}
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	if logFormat == logFormatJson {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	}
	logger = level.NewFilter(logger, lvl)
	return log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
}

func main() {
	app := extkingpin.NewApp(kingpin.New(filepath.Base(os.Args[0]), "Control state of your deployments.").Version(version.Version))
	logLevel := app.Flag("log.level", "Log filtering level.").
		Default("info").Enum("error", "warn", "info", "debug")
	logFormat := app.Flag("log.format", "Log format to use. Possible options: logfmt or json.").
		Default(logFormatLogfmt).Enum(logFormatLogfmt, logFormatJson)

	defCacheDir, _ := os.UserCacheDir()
	cacheDir := app.Flag("cache.dir", "Path for cache (e.g git repo checkouts).").Default(filepath.Join(defCacheDir, "statectl")).String()

	// TODO(bwplotka): Auto-gen docs.
	cfgPath := app.Flag("project.config-file", "Path for YAML configuration for project statectl file. All commands relates to certain project. See config.Project for format.").Default("./.statectl.yaml").String()

	registerPropose(app)
	registerDiff(app)

	cmd, runner := app.Parse()
	logger := setupLogger(*logLevel, *logFormat)

	var g run.Group
	{
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			b, err := ioutil.ReadFile(*cfgPath)
			if err != nil {
				return errors.Wrapf(err, "read file %v", *cfgPath)
			}

			cfg, err := loader.LoadProjectConfig(b)
			if err != nil {
				return errors.Wrapf(err, "load project from %v", *cfgPath)
			}

			p, err := project.New(ctx, logger, cfg, *cacheDir)
			if err != nil {
				return errors.Wrapf(err, "new project from %v", *cfgPath)
			}
			return runner(p)
		}, func(err error) { cancel() })
	}

	// Listen for termination signals.
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(logger, cancel)
		}, func(error) {
			close(cancel)
		})
	}

	if err := g.Run(); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		fmt.Printf("%+v\n", errors.Wrapf(err, "%s command failed", cmd))
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "exiting")
}

func interrupt(logger log.Logger, cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-c:
		level.Info(logger).Log("msg", "caught signal. Exiting.", "signal", s)
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}
