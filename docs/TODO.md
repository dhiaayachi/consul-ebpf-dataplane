
## TODOs
- [x] Mount `/sys/fs/cgroup` in the connect-inject container
- [x] Add code to load the ebpf program into the connect-inject container at start.
- [] Modify envoy bootstrap to use a VIP instead of consul-dataplane address
- [] Add code to inject the HashMap entry when a new sidecar is being created (VIP-->Server Address)

## Stretch
- Add code to unload the eBPF program when deleting he connect-inject container