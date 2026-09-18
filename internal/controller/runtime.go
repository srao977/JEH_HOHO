// File: runtime.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 implementation
// Product/Component: JEH-HOHO / Production composition root
// Purpose: Compose and supervise the complete product without embedding business logic.
// Responsibilities: Preflight; source, DSE_JEH, persistence, RPC, Viewer start; status; stop.
// Inputs/Outputs: Package configuration and credentials in; owned product runtime out.
// Configuration: Package-relative JSON loaded by LoadConfig.
// Dependencies: Graduated source, DSE_JEH, Mongo writers, generated services, packaged Node.
// Invariants: One source path; READY needs no first Bar; shutdown reverses owned components.
// Failure behavior: Component failures become plain FAILED/DEGRADED status and propagate.
// Concurrency/Lifecycle: Controller owns all goroutines, listeners, files, and child handles.
// Non-responsibilities: JEH math, Capital behavior, execution policy, replay, or paper trading.

package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	"jeh-hoho/internal/jeh/app"
	jehconfig "jeh-hoho/internal/jeh/config"
	"jeh-hoho/internal/jeh/execution"
	dsejehv1 "jeh-hoho/internal/jeh/gen/dse_jeh/v1"
	productsource "jeh-hoho/internal/jeh/input/product"
	"jeh-hoho/internal/lifecycle"
	"jeh-hoho/internal/source"
	"jeh-hoho/internal/source/alpaca"
	sourcepersistence "jeh-hoho/internal/source/persistence"

	"google.golang.org/grpc"
)

// Run loads and operates one complete product instance until cancellation or STOP.
func Run(ctx context.Context, home string, output io.Writer) error {
	config, credentials, err := LoadConfig(home)
	if err != nil {
		return fmt.Errorf("PREFLIGHT FAILED: %w", err)
	}
	fmt.Fprintln(output, "JEH-HOHO STARTING")
	fmt.Fprintln(output, "JEH-HOHO PREFLIGHT")
	preflightContext, cancelPreflight := context.WithTimeout(ctx, 20*time.Second)
	err = Preflight(preflightContext, config, credentials)
	cancelPreflight()
	if err != nil {
		return fmt.Errorf("PREFLIGHT FAILED: %w", err)
	}
	return runConfigured(ctx, config, credentials, output)
}

func runConfigured(ctx context.Context, config Config, credentials Credentials, output io.Writer) error {
	return runConfiguredWith(ctx, config, credentials, output, runtimeDependencies{startViewer: startViewer})
}

type viewerRuntime interface {
	Errors() <-chan error
	Stop()
}

type runtimeDependencies struct {
	traceWriter     execution.TraceWriter
	reservoirWriter execution.CapitalReservoirWriter
	startViewer     func(Config, io.Writer) (viewerRuntime, error)
}

func runConfiguredWith(ctx context.Context, config Config, credentials Credentials, output io.Writer, dependencies runtimeDependencies) error {
	started := time.Now().UTC()
	collectionRunID := started.Format("20060102T150405.000000000Z")
	pipelineRunID := "pipeline-" + collectionRunID
	token, err := newToken()
	if err != nil {
		return fmt.Errorf("create product ownership token: %w", err)
	}
	_ = os.Remove(stopPath(config.Home))

	sourceWriter, err := sourcepersistence.Open(config.SourceDataDirectory, collectionRunID, "accepted")
	if err != nil {
		return fmt.Errorf("open source persistence: %w", err)
	}
	stream := alpaca.NewStream(config.AlpacaStreamURL, config.AlpacaFeed, alpaca.Credentials{Key: credentials.AlpacaKey, Secret: credentials.AlpacaSecret})
	engine, err := source.NewEngine(source.EngineConfig{CollectionRunID: collectionRunID, Feed: config.AlpacaFeed, Symbols: config.Symbols, BufferCapacity: config.SourceBufferCapacity, Source: stream, Writer: sourceWriter})
	if err != nil {
		_ = sourceWriter.Close()
		return err
	}
	sourceListener, err := net.Listen("tcp", config.SourceAddress)
	if err != nil {
		_ = sourceWriter.Close()
		return fmt.Errorf("start Bar Sequence Source: %w", err)
	}
	config.SourceAddress = sourceListener.Addr().String()
	sourceServer := grpc.NewServer()
	jehhohov1.RegisterAcceptedBarSourceServiceServer(sourceServer, source.NewServer(engine.Accepted()))
	sourceContext, cancelSource := context.WithCancel(context.Background())
	sourceServeErrors := make(chan error, 1)
	sourceEngineErrors := make(chan error, 1)
	go func() { sourceServeErrors <- sourceServer.Serve(sourceListener) }()
	go func() { sourceEngineErrors <- engine.Run(sourceContext) }()

	jehOptions := app.Options{Source: productsource.New(config.SourceAddress, "DSE_JEH"), TraceWriter: dependencies.traceWriter, ReservoirWriter: dependencies.reservoirWriter}
	jehConfiguration := jehconfig.Config{
		Mode: dsejehv1.RuntimeMode_RUNTIME_MODE_ONLINE, MongoURI: config.MongoURI, MongoDatabase: config.MongoDatabase,
		MongoTraceCollection: config.MongoTraceCollection, MongoReservoirCollection: config.MongoReservoirCollection,
		CollectionRunID: collectionRunID, PipelineRunID: pipelineRunID, RunType: execution.RunType(config.RunType),
		StartingCapital: config.StartingCapital, AllocationPct: config.AllocationPercent, Risk: config.Risk,
		GRPCAddress: config.SourceAddress, ServerAddress: config.JEHAddress, StatusInterval: 5 * time.Second, RemainRunning: true, Symbols: config.Symbols,
	}
	application, err := app.NewProduct(ctx, jehConfiguration, jehOptions)
	if err != nil {
		sourceServer.Stop()
		cancelSource()
		_ = sourceWriter.Close()
		return fmt.Errorf("start DSE_JEH persistence: %w", err)
	}
	identity := &jehhohov1.ProductIdentity{ProductVersion: config.ProductVersion, ProductRunId: application.RuntimeID(), StartedUnixMs: started.UnixMilli()}
	statusStore := lifecycle.NewStore(identity, config.Symbols)
	statusStore.Bind(stream.Health, application.Snapshot)
	application.RegisterProductOperations(lifecycle.NewOperations(statusStore.Snapshot))
	viewerURL := "http://" + config.ViewerAddress
	record := runtimeRecord{PID: os.Getpid(), Executable: executablePath(), ProductRunID: identity.ProductRunId, State: "CONNECTING", ViewerURL: viewerURL, StartedUTC: started.Format(time.RFC3339), Token: token}
	if err := writeRecord(config.Home, record); err != nil {
		sourceServer.Stop()
		cancelSource()
		_ = sourceWriter.Close()
		return err
	}
	defer clearRuntime(config.Home)
	statusStore.Transition(jehhohov1.ProductLifecycleState_PRODUCT_LIFECYCLE_STATE_CONNECTING, "connecting to Alpaca")
	fmt.Fprintln(output, "JEH-HOHO CONNECTING")

	jehContext, cancelJEH := context.WithCancel(context.Background())
	jehErrors := make(chan error, 1)
	go func() { jehErrors <- application.Run(jehContext) }()
	if err := waitForTCP(ctx, config.JEHAddress, 10*time.Second); err != nil {
		cancelJEH()
		sourceServer.Stop()
		cancelSource()
		_ = sourceWriter.Close()
		return fmt.Errorf("DSE_JEH failed to start: %w", err)
	}
	viewer, err := dependencies.startViewer(config, output)
	if err != nil {
		cancelJEH()
		<-jehErrors
		sourceServer.Stop()
		cancelSource()
		_ = sourceWriter.Close()
		return err
	}
	viewerErrors := viewer.Errors()
	statusStore.Viewer(true)

	stopTicker := time.NewTicker(250 * time.Millisecond)
	statusTicker := time.NewTicker(time.Second)
	defer stopTicker.Stop()
	defer statusTicker.Stop()
	ready := false
	sourceEngineDone := false
	var runErr error
	for runErr == nil {
		select {
		case <-ctx.Done():
			runErr = ctx.Err()
		case err := <-jehErrors:
			if err != nil {
				runErr = fmt.Errorf("DSE_JEH stopped: %w", err)
			} else {
				runErr = errors.New("DSE_JEH stopped unexpectedly")
			}
		case err := <-sourceServeErrors:
			if err != nil && !errors.Is(err, grpc.ErrServerStopped) && !errors.Is(err, context.Canceled) {
				runErr = fmt.Errorf("Bar Sequence Source stopped: %w", err)
			}
		case err := <-sourceEngineErrors:
			sourceEngineDone = true
			if err != nil && !errors.Is(err, context.Canceled) {
				runErr = fmt.Errorf("Alpaca Bar Sequence Source stopped: %w", err)
			}
		case err := <-viewerErrors:
			if err != nil {
				runErr = fmt.Errorf("Viewer stopped: %w", err)
			} else {
				runErr = errors.New("Viewer stopped unexpectedly")
			}
		case <-stopTicker.C:
			if stopRequested(config.Home, token) {
				runErr = context.Canceled
			}
		case <-statusTicker.C:
			health := stream.Health()
			if !ready && health.State == "healthy" {
				ready = true
				statusStore.Transition(jehhohov1.ProductLifecycleState_PRODUCT_LIFECYCLE_STATE_READY, "")
				record.State, record.Reason = "READY", ""
				_ = writeRecord(config.Home, record)
				fmt.Fprintln(output, "JEH-HOHO READY")
				fmt.Fprintln(output, "Viewer:", viewerURL)
				if config.OpenViewer {
					openBrowser(viewerURL)
				}
			} else if ready && health.State != "healthy" {
				ready = false
				statusStore.Transition(jehhohov1.ProductLifecycleState_PRODUCT_LIFECYCLE_STATE_DEGRADED, "Alpaca is reconnecting; continuity is unknown")
				record.State, record.Reason = "DEGRADED", "Alpaca is reconnecting; continuity is unknown"
				_ = writeRecord(config.Home, record)
				fmt.Fprintln(output, "JEH-HOHO DEGRADED: Alpaca is reconnecting")
			}
		}
	}

	statusStore.Transition(jehhohov1.ProductLifecycleState_PRODUCT_LIFECYCLE_STATE_STOPPING, "shutdown requested")
	record.State, record.Reason = "STOPPING", "shutdown requested"
	_ = writeRecord(config.Home, record)
	fmt.Fprintln(output, "JEH-HOHO STOPPING")
	viewer.Stop()
	statusStore.Viewer(false)
	cancelJEH()
	select {
	case <-jehErrors:
	case <-time.After(10 * time.Second):
	}
	sourceServer.GracefulStop()
	cancelSource()
	if !sourceEngineDone {
		select {
		case <-sourceEngineErrors:
		case <-time.After(10 * time.Second):
		}
	}
	_ = sourceWriter.Close()
	record.State, record.Reason = "STOPPED", ""
	_ = writeRecord(config.Home, record)
	fmt.Fprintln(output, "JEH-HOHO STOPPED")
	if errors.Is(runErr, context.Canceled) {
		return nil
	}
	return runErr
}

type ownedViewer struct {
	command *exec.Cmd
	errors  chan error
}

func (viewer *ownedViewer) Errors() <-chan error { return viewer.errors }

func (viewer *ownedViewer) Stop() {
	if viewer == nil || viewer.command == nil || viewer.command.Process == nil || viewer.command.ProcessState != nil {
		return
	}
	_ = viewer.command.Process.Signal(os.Interrupt)
	select {
	case <-viewer.errors:
		return
	case <-time.After(3 * time.Second):
		_ = viewer.command.Process.Kill()
	}
}

func startViewer(config Config, output io.Writer) (viewerRuntime, error) {
	_, port, _ := net.SplitHostPort(config.ViewerAddress)
	command := exec.Command(config.NodeExecutable, "server.js")
	command.Dir = config.ViewerDirectory
	command.Env = append(os.Environ(), "HOSTNAME=127.0.0.1", "PORT="+port, "JEH_HOHO_SERVER_ADDRESS="+config.JEHAddress, "JEH_HOHO_MONGO_URI="+config.MongoURI, "JEH_HOHO_MONGO_DB="+config.MongoDatabase, "JEH_HOHO_MONGO_COLLECTION="+config.MongoReservoirCollection)
	command.Stdout, command.Stderr = output, output
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start packaged Viewer: %w", err)
	}
	errors := make(chan error, 1)
	go func() { errors <- command.Wait() }()
	viewer := &ownedViewer{command: command, errors: errors}
	if err := waitForHTTP("http://"+config.ViewerAddress, 15*time.Second); err != nil {
		viewer.Stop()
		return nil, fmt.Errorf("packaged Viewer failed to start: %w", err)
	}
	return viewer, nil
}

func waitForTCP(ctx context.Context, address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 150*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return fmt.Errorf("listener %s was not ready", address)
}

func waitForHTTP(address string, timeout time.Duration) error {
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response, err := client.Get(address)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("Viewer %s was not ready", address)
}

func executablePath() string { path, _ := os.Executable(); return path }
func openBrowser(address string) {
	if strings.EqualFold(filepath.Ext(os.Getenv("COMSPEC")), ".exe") {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", address).Start()
	}
}
