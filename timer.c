//go:build ignore

#include <common.h>
#include <bpf_helpers.h>

// flags allowed in bpf_timer_init
#define CLOCK_REALTIME 0
#define CLOCK_MONOTONIC 1
#define CLOCK_BOOTTIME 7

#define EBUSY 0x10

char __license[] SEC("license") = "Dual MIT/GPL";

struct mapval {
  struct bpf_timer timer;
  u64 cb_called;
};

// we use a map to store the timer
// this is an array of 1 element
struct {
  __uint(type, BPF_MAP_TYPE_ARRAY);
  __type(key, u32);
  __type(value, struct mapval);
  __uint(max_entries, 1);
} timer_map SEC(".maps");

// callback for the timer
static int timer_cb(void *map, u32 *key, struct mapval *v) {
  long ret;
  u64 called;

  called = __atomic_fetch_add(&v->cb_called, 1, __ATOMIC_RELAXED);
  bpf_printk("timer_callback: %u", called);

  // start the timer again
  // it will call the callback after 1 second
  // second parameter is the interval in nanoseconds
  ret = bpf_timer_start(&v->timer, 1000000000llu, 0);
  if (ret) {
    bpf_printk("error: timer_callback: timer_start: %ld", ret);
    return 0;
  }

  return 0;
}

SEC("syscall")
int start_timer(void *ctx) {
  struct mapval *v;
  int zero = 0;
  long ret;

  // get the value from the map that contains the timer
  v = bpf_map_lookup_elem(&timer_map, &zero);
  if (!v) {
    bpf_printk("error: no map value");
    return 1;
  }

  // initialize the timer
  ret = bpf_timer_init(&v->timer, &timer_map, CLOCK_MONOTONIC);
  if (ret == -EBUSY) {
    bpf_printk("info: timer already initialized");
    return 0;
  } else if (ret) {
    bpf_printk("error: timer_init: %ld", ret);
    return ret;
  }

  // set the callback for the timer
  ret = bpf_timer_set_callback(&v->timer, timer_cb);
  if (ret) {
    bpf_printk("error: timer_set_callback: %ld", ret);
    return ret;
  }

  // start the timer
  // it will call the callback after 1 second
  // second parameter is the interval in nanoseconds
  ret = bpf_timer_start(&v->timer, 1000000000llu, 0);
  if (ret) {
    bpf_printk("error: timer_start: %ld", ret);
    return ret;
  }

  bpf_printk("info: timer initialized");

  return 0;
}
