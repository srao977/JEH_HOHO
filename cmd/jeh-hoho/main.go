// File: main.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 production entry point
// Product/Component: JEH-HOHO / Product controller executable
// Purpose: Provide one operator START, STOP, and STATUS entry point.
// Responsibilities: Resolve package home, dispatch commands, and handle console shutdown.
// Inputs/Outputs: Operator command in; plain lifecycle status and exit code out.
// Configuration: Package-relative config files; optional --home for validation only.
// Dependencies: Internal product controller.
// Invariants: Contains no JEH, execution, or Capital business logic.
// Failure behavior: Prints one concise error without a Go stack trace.
// Concurrency/Lifecycle: START handles Ctrl+C and delegates owned shutdown.
// Non-responsibilities: Generated contracts, persistence logic, or Viewer rendering.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"jeh-hoho/internal/controller"
)

func main() {
	home := productHome()
	command := "start"
	for index := 1; index < len(os.Args); index++ {
		if os.Args[index] == "--home" && index+1 < len(os.Args) {
			home = os.Args[index+1]
			index++
			continue
		}
		command = strings.ToLower(os.Args[index])
	}
	var err error
	switch command {
	case "start":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		err = controller.Run(ctx, home, os.Stdout)
	case "stop":
		err = controller.RequestStop(home)
		if err == nil {
			fmt.Println("JEH-HOHO stop requested")
		}
	case "status":
		var status string
		status, err = controller.ReadOperatorStatus(home)
		if err == nil {
			fmt.Println(status)
		}
	case "support":
		var path string
		path, err = controller.CreateSupportReport(home)
		if err == nil {
			fmt.Println("Support report created:", path)
		}
	default:
		err = fmt.Errorf("unknown command %q; use start, status, stop, or support", command)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "JEH-HOHO:", err)
		os.Exit(1)
	}
}

func productHome() string {
	executable, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(filepath.Dir(executable))
}
