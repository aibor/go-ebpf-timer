// Sample output:
//
// $ go generate  ./... &&  go run -exec sudo .
// Compiled /home/apapag/go-ebpf/bpf_bpfel.o
// Stripped /home/apapag/go-ebpf/bpf_bpfel.o
// Wrote /home/apapag/go-ebpf/bpf_bpfel.go
// Compiled /home/apapag/go-ebpf/bpf_bpfeb.o
// Stripped /home/apapag/go-ebpf/bpf_bpfeb.o
// Wrote /home/apapag/go-ebpf/bpf_bpfeb.go
// 2024/03/12 13:15:47 Comm
// 2024/03/12 13:15:49 fcntl
// 2024/03/12 13:15:54 fcntl

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -type event bpf fentry.c -- -I./headers

func main() {
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading objects: %v", err)
	}
	defer objs.Close()

	if err := objs.bpfMaps.TimerMap.Pin("/sys/fs/bpf/timer_map"); err != nil {
		log.Fatalf("pinning timer map: %v", err)
	}

	zero := uint32(0)
	value := bpfMapval{Initialized: 0}
	if err := objs.bpfMaps.TimerMap.Update(&zero, &value, 0); err != nil {
		log.Fatalf("initializing timer map: %v", err)
	}

	link, err := link.AttachTracing(
		link.TracingOptions{
			Program:    objs.bpfPrograms.SecurityFileFcntl,
			AttachType: ebpf.AttachTraceFEntry,
		})
	if err != nil {
		log.Fatal(err)
	}
	defer link.Close()

	rd, err := ringbuf.NewReader(objs.bpfMaps.Events)
	if err != nil {
		log.Fatalf("opening ringbuf reader: %s", err)
	}
	defer rd.Close()

	go func() {
		<-stopper

		if err := rd.Close(); err != nil {
			log.Fatalf("closing ringbuf reader: %s", err)
		}

		if err := os.Remove("/sys/fs/bpf/timer_map"); err != nil {
			log.Fatalf("removing timer map: %v", err)
		}
	}()

	log.Printf("%-16s %-8s %-8s", "Comm", "PID", "TGID")

	// bpfEvent is generated by bpf2go.
	var event bpfEvent
	for {
		record, err := rd.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				log.Println("received signal, exiting..")
				return
			}
			log.Printf("reading from reader: %s", err)
			continue
		}

		// Parse the ringbuf event entry into a bpfEvent structure.
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			log.Printf("parsing ringbuf event: %s", err)
			continue
		}

		log.Printf("%-16s %-8s %-8s", event.Comm, strconv.FormatUint(uint64(event.Pid), 10), strconv.FormatUint(uint64(event.Tgid), 10))
	}
}
