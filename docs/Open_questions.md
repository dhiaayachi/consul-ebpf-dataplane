# Open Questions

- How to mount /sys/fs/cgroup in the controller Pod
- What is the best way to run the controller in each host (a DeamonSet?)
- How to intercept the event of deploying a new sidecar and inject the relevant VIP in the HashMap, I guess this need to be done in all hosts?

# Random Thoughts
- Maybe for simplicity we can inject the VIP in the init container of the service Pod. That way it applies to the host it's initially deployed in. We don't need to care of it being evicted for the PoC.
- As a starting point we can deploy a single node K8S and run the binary to redirect to the servers manually and hack consul-control-plane (or maybe `consul connect envoy` command?) to inject the VIP