// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package auto provides OpenTelemetry automatic tracing instrumentation for Go
// packages using eBPF.
package auto

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"

	"go.opentelemetry.io/auto/internal/pkg/instrumentation"
	dbSql "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/database/sql"
	kafkaConsumer "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/github.com/segmentio/kafka-go/consumer"
	kafkaProducer "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/github.com/segmentio/kafka-go/producer"
	autosdk "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/go.opentelemetry.io/auto/sdk"
	otelTraceGlobal "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/go.opentelemetry.io/otel/traceglobal"
	grpcClient "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/google.golang.org/grpc/client"
	grpcServer "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/google.golang.org/grpc/server"
	httpClient "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/net/http/client"
	httpServer "go.opentelemetry.io/auto/internal/pkg/instrumentation/bpf/net/http/server"
	"go.opentelemetry.io/auto/internal/pkg/instrumentation/probe"
	"go.opentelemetry.io/auto/internal/pkg/process"
	"go.opentelemetry.io/auto/pipeline"
	"go.opentelemetry.io/auto/pipeline/otelsdk"
)

// envLogLevelKey is the key for the environment variable value containing the log level.
const envLogLevelKey = "OTEL_LOG_LEVEL"

// Instrumentation manages and controls all OpenTelemetry Go
// auto-instrumentation.
type Instrumentation struct {
	manager *instrumentation.Manager
	cleanup func()

	stopMu  sync.Mutex
	stop    context.CancelFunc
	stopped chan struct{}
}

// NewInstrumentation returns a new [Instrumentation] configured with the
// provided opts.
//
// If conflicting or duplicate options are provided, the last one will have
// precedence and be used.
func NewInstrumentation(
	ctx context.Context,
	opts ...InstrumentationOption,
) (*Instrumentation, error) {
	c, err := newInstConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := c.validate(); err != nil {
		return nil, err
	}

	p := []probe.Probe{
		grpcClient.New(c.logger, Version()),
		grpcServer.New(c.logger, Version()),
		httpServer.New(c.logger, Version()),
		httpClient.New(c.logger, Version()),
		dbSql.New(c.logger, Version()),
		kafkaProducer.New(c.logger, Version()),
		kafkaConsumer.New(c.logger, Version()),
		autosdk.New(c.logger),
		otelTraceGlobal.New(c.logger),
	}

	cp := convertConfigProvider(c.cp)
	mngr, err := instrumentation.NewManager(c.logger, c.handler, c.pid, cp, p...)
	if err != nil {
		return nil, err
	}

	return &Instrumentation{manager: mngr, cleanup: c.handlerClose}, nil
}

// Load loads and attaches the relevant probes to the target process.
func (i *Instrumentation) Load(ctx context.Context) error {
	return i.manager.Load(ctx)
}

// Run starts the instrumentation. It must be called after [Instrumentation.Load].
//
// This function will not return until either ctx is done, an unrecoverable
// error is encountered, or Close is called.
func (i *Instrumentation) Run(ctx context.Context) error {
	if i.cleanup != nil {
		defer i.cleanup()
	}

	ctx, err := i.newStop(ctx)
	if err != nil {
		return err
	}

	err = i.manager.Run(ctx)
	close(i.stopped)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func (i *Instrumentation) newStop(parent context.Context) (context.Context, error) {
	i.stopMu.Lock()
	defer i.stopMu.Unlock()

	if i.stop != nil {
		return parent, errors.New("instrumentation already running")
	}

	ctx, stop := context.WithCancel(parent)
	i.stop, i.stopped = stop, make(chan struct{})
	return ctx, nil
}

// Close closes the Instrumentation, cleaning up all used resources.
func (i *Instrumentation) Close() error {
	i.stopMu.Lock()
	defer i.stopMu.Unlock()

	if i.stop == nil {
		// if stop is not set, the instrumentation is not running
		// stop the manager to clean up resources
		return i.manager.Stop()
	}

	if i.cleanup != nil {
		defer i.cleanup()
	}

	i.stop()
	<-i.stopped
	i.stop, i.stopped = nil, nil

	return nil
}

// InstrumentationOption applies a configuration option to [Instrumentation].
type InstrumentationOption interface {
	apply(context.Context, instConfig) (instConfig, error)
}

type instConfig struct {
	pid          process.ID
	handler      *pipeline.Handler
	handlerClose func()
	logger       *slog.Logger
	sampler      Sampler
	cp           ConfigProvider
}

func newInstConfig(ctx context.Context, opts []InstrumentationOption) (instConfig, error) {
	c := instConfig{pid: -1}
	var err error
	for _, opt := range opts {
		if opt != nil {
			var e error
			c, e = opt.apply(ctx, c)
			err = errors.Join(err, e)
		}
	}

	// Defaults.
	if c.handler == nil {
		attrs := []attribute.KeyValue{
			semconv.TelemetryDistroVersionKey.String(Version()),
		}

		// Add additional process information for the target.
		var e error
		bi, e := c.pid.BuildInfo()
		if e == nil {
			attrs = append(attrs, semconv.ProcessRuntimeVersion(bi.GoVersion))

			var compiler string
			for _, setting := range bi.Settings {
				if setting.Key == "-compiler" {
					compiler = setting.Value
					break
				}
			}
			switch compiler {
			case "":
				// Ignore empty.
			case "gc":
				attrs = append(attrs, semconv.ProcessRuntimeName("go"))
			default:
				attrs = append(attrs, semconv.ProcessRuntimeName(compiler))
			}
		}

		th, e := otelsdk.NewTraceHandler(
			ctx,
			otelsdk.WithEnv(),
			otelsdk.WithResourceAttributes(attrs...),
		)
		err = errors.Join(err, e)

		if th != nil {
			c.handler = &pipeline.Handler{TraceHandler: th}

			c.handlerClose = sync.OnceFunc(func() {
				ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
				defer stop()

				if err := th.Shutdown(ctx); err != nil {
					c.logger.Error("failed cleanup", "error", err)
				}
			})
		}
	}
	if c.sampler == nil {
		c.sampler = DefaultSampler()
	}

	if c.logger == nil {
		c.logger = newLogger(nil)
	}

	if c.cp == nil {
		c.cp = newNoopConfigProvider(c.sampler)
	}

	return c, err
}

func (c instConfig) validate() error {
	return c.pid.Validate()
}

// newLogger is used for testing.
var newLogger = newLoggerFunc

func newLoggerFunc(level slog.Leveler) *slog.Logger {
	opts := &slog.HandlerOptions{AddSource: true, Level: level}
	h := slog.NewJSONHandler(os.Stderr, opts)
	return slog.New(h)
}

type fnOpt func(context.Context, instConfig) (instConfig, error)

func (o fnOpt) apply(ctx context.Context, c instConfig) (instConfig, error) { return o(ctx, c) }

// WithPID returns an [InstrumentationOption] defining the target binary for
// [Instrumentation] that is being run with the provided PID.
//
// If multiple of these options are provided to an [Instrumentation], the last
// one will be used.
func WithPID(pid int) InstrumentationOption {
	return fnOpt(func(_ context.Context, c instConfig) (instConfig, error) {
		c.pid = process.ID(pid)
		return c, nil
	})
}

var lookupEnv = os.LookupEnv

// WithEnv returns an [InstrumentationOption] that will configure
// [Instrumentation] using the values defined by the following environment
// variables:
//
//   - OTEL_LOG_LEVEL: sets the default logger's minimum logging level
//   - OTEL_TRACES_SAMPLER: sets the trace sampler
//   - OTEL_TRACES_SAMPLER_ARG: optionally sets the trace sampler argument
//
// This option may conflict with [WithSampler] if their respective environment
// variable is defined. If more than one of these options are used, the last
// one provided to an [Instrumentation] will be used.
//
// If [WithLogger] is used, OTEL_LOG_LEVEL will not be used for the
// [Instrumentation] logger. Instead, the [slog.Logger] passed to that option
// will be used as-is.
//
// If [WithLogger] is not used, OTEL_LOG_LEVEL will be parsed and the default
// logger used by the configured [Instrumentation] will use that level as its
// minimum logging level.
func WithEnv() InstrumentationOption {
	return fnOpt(func(ctx context.Context, c instConfig) (instConfig, error) {
		var err error
		if val, ok := lookupEnv(envLogLevelKey); c.logger == nil && ok {
			var level slog.Level
			if e := level.UnmarshalText([]byte(val)); e != nil {
				e = fmt.Errorf("parse log level %q: %w", val, e)
				err = errors.Join(err, e)
			} else {
				c.logger = newLogger(level)
			}
		}
		if s, e := newSamplerFromEnv(lookupEnv); e != nil {
			err = errors.Join(err, e)
		} else {
			c.sampler = s
		}
		return c, err
	})
}

// WithSampler returns an [InstrumentationOption] that will configure
// an [Instrumentation] to use the provided sampler to sample OpenTelemetry traces.
//
// This currently is a no-op. It is expected to take effect in the next release.
func WithSampler(sampler Sampler) InstrumentationOption {
	return fnOpt(func(_ context.Context, c instConfig) (instConfig, error) {
		c.sampler = sampler
		return c, nil
	})
}

// WithLogger returns an [InstrumentationOption] that will configure an
// [Instrumentation] to use the provided logger.
//
// If this option is used and [WithEnv] is also used, OTEL_LOG_LEVEL is ignored
// by the configured [Instrumentation]. This passed logger takes precedence and
// is used as-is.
//
// If this option is not used, the [Instrumentation] will use an [slog.Loogger]
// backed by an [slog.JSONHandler] outputting to STDERR as a default.
func WithLogger(logger *slog.Logger) InstrumentationOption {
	return fnOpt(func(_ context.Context, c instConfig) (instConfig, error) {
		c.logger = logger
		return c, nil
	})
}

// WithConfigProvider returns an [InstrumentationOption] that will configure
// an [Instrumentation] to use the provided ConfigProvider. The ConfigProvider
// is used to provide the initial configuration and update the configuration of
// the instrumentation in runtime.
func WithConfigProvider(cp ConfigProvider) InstrumentationOption {
	return fnOpt(func(_ context.Context, c instConfig) (instConfig, error) {
		c.cp = cp
		return c, nil
	})
}

// WithHandler returns an [InstrumentationOption] that will configure an
// [Instrumentation] to use h to handle generated telemetry.
//
// If this options is not used, the Handler returned from [otelsdk.NewHandler] with
// environment configuration will be used.
func WithHandler(h *pipeline.Handler) InstrumentationOption {
	return fnOpt(func(_ context.Context, c instConfig) (instConfig, error) {
		if h == nil {
			return c, errors.New("nil handler")
		}
		c.handler = h
		return c, nil
	})
}
