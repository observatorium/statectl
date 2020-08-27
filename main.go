// Copyright (c) The Observatorium Authors.
// Licensed under the Apache License 2.0.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/observatorium/statectl/pkg/extkingpin"
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

	registerPropose(app)

	cmd, setup := app.Parse()
	logger := setupLogger(*logLevel, *logFormat)

	var g run.Group

	if err := setup(&g, logger); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		level.Error(logger).Log("err", fmt.Sprintf("%+v", errors.Wrapf(err, "preparing %s command failed", cmd)))
		os.Exit(1)
	}
	// Dummy actor to immediately kill the group after the run function returns.
	g.Add(func() error { return nil }, func(error) {})
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
		level.Error(logger).Log("err", fmt.Sprintf("%+v", errors.Wrapf(err, "%s command failed", cmd)))
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

func reload(logger log.Logger, cancel <-chan struct{}, r chan<- struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	for {
		select {
		case s := <-c:
			level.Info(logger).Log("msg", "caught signal. Reloading.", "signal", s)
			select {
			case r <- struct{}{}:
				level.Info(logger).Log("msg", "reload dispatched.")
			default:
			}
		case <-cancel:
			return errors.New("canceled")
		}
	}
}

func getFlagsMap(flags []*kingpin.FlagModel) map[string]string {
	flagsMap := map[string]string{}

	// Exclude kingpin default flags to expose only Thanos ones.
	boilerplateFlags := kingpin.New("", "").Version("")

	for _, f := range flags {
		if boilerplateFlags.GetFlag(f.Name) != nil {
			continue
		}
		flagsMap[f.Name] = f.Value.String()
	}

	return flagsMap
}
