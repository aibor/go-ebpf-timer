# go-ebpf-timer

This is a simple set of programs to demonstrate the use of eBPF timers. It uses
[cilium/ebpf](https://github.com/cilium/ebpf) to interract with eBPF subsystem,
and is based on the project
[tpapagian/go-ebpf-timer](https://github.com/tpapagian/go-ebpf-timer).

The high-level idea is that we will initialize a timer by using a `syscall`
program. This will call a function every second until we terminate the main
program. We make sure that the timer will be only initialized once even if the
`syscall` program is triggered more than once.

In order to trigger our eBPF program, we just need to use `BPF_PROG_RUN`.

## Example output:

The program loads the BPF program, calls it twice, and prints BPF tracing 
output for 10 seconds.

```console
$ go run -exec "go run github.com/aibor/virtrun -kernel /boot/vmlinuz-linux -transport isa" .
# tracer: nop
#
# entries-in-buffer/entries-written: 0/0   #P:4
#
#                                _-----=> irqs-off/BH-disabled
#                               / _----=> need-resched
#                              | / _---=> hardirq/softirq
#                              || / _--=> preempt-depth
#                              ||| / _-=> migrate-disable
#                              |||| /     delay
#           TASK-PID     CPU#  |||||  TIMESTAMP  FUNCTION
#              | |         |   |||||     |         |

            main-92      [000] ...11     0.276868: bpf_trace_printk: info: timer initialized
            main-92      [000] ...11     0.276879: bpf_trace_printk: info: timer already initialized
          <idle>-0       [000] ..s2.     1.276894: bpf_trace_printk: timer_callback: 0
          <idle>-0       [000] ..s2.     2.276929: bpf_trace_printk: timer_callback: 0
          <idle>-0       [000] ..s2.     3.276980: bpf_trace_printk: timer_callback: 0
          <idle>-0       [000] ..s2.     4.277020: bpf_trace_printk: timer_callback: 0
          <idle>-0       [000] ..s2.     5.277050: bpf_trace_printk: timer_callback: 0
          <idle>-0       [000] ..s2.     6.277087: bpf_trace_printk: timer_callback: 0
          <idle>-0       [000] ..s2.     7.277115: bpf_trace_printk: timer_callback: 0
          <idle>-0       [000] ..s2.     8.277149: bpf_trace_printk: timer_callback: 0

```

## Test Platform

This is tested with a Arch Linux kernel 6.12.1 using
[virtrun](https://github.com/aibor/virtrun).
