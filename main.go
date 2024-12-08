package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target bpfel bpf timer.c -- -I./headers

func run() error {
	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		return fmt.Errorf("loading objects: %w", err)
	}
	defer objs.Close()

	// Setup tracing reader.
	tracingDir := "/sys/kernel/tracing/"

	traceHeader, err := os.ReadFile(tracingDir + "trace")
	if err != nil {
		return fmt.Errorf("read trace: %w", err)
	}

	tracePipe, err := os.Open(tracingDir + "trace_pipe")
	if err != nil {
		return fmt.Errorf("open trace pipe: %w", err)
	}
	defer tracePipe.Close()

	go func() {
		fmt.Println(string(traceHeader))
		_, err := io.Copy(os.Stdout, tracePipe)
		if err != nil && !errors.Is(err, os.ErrClosed) {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}()

	// Run program twice to see if the init guard works.
	for range 2 {
		ret, err := objs.StartTimer.Run(&ebpf.RunOptions{})
		if err == nil && ret != 0 {
			err = syscall.Errno(-ret)
			// Wait a bit to give tracing some time to print error messages.
			time.Sleep(10 * time.Millisecond)
		}
		if err != nil {
			return fmt.Errorf("prog run: %w", err)
		}
	}

	time.Sleep(10 * time.Second)

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
