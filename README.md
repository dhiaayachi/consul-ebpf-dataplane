# Consul eBPF DataPlane
## Backgroup
Today when deploying Consul in agentless mode in a kubernetes environment as a service-mesh, a consul-dataplane container is injected in each pod. The responsibility of consul-dataplane, is:

Finding the servers addresses (using go-discover, dns...)
Ensuring the connectivity of the envoy sidecar to the server by filling the role of an xDS proxy
proxying the gRPC requests to the servers and ensure even load between servers.
proxying DNS requests to the servers

## Idea
The idea proposed here is to build a PoC that demonstrate the possibility of replacing the current architecture with an eBPF program and a single controller per host that pilot the eBPF program using a hashmap.

The controller will be responsible of:

Collecting the Consul servers IPs
Injecting the server IPs into a kernel hashmap using a VIP per Pod as key and selecting a server to connect to per Pod
Update the map when the list of Consul servers is updates
Injecting the VIP into each envoy bootstrap Config
The eBPF program responsibility will be:

When an envoy container connect to a VIP, replace the socket IP with a server IP fetched from the kernel hashmap
Optionally, report number of connections per server.
The advantage of this approach is to optimize the Consul in Kubernetes deployment in agentless context by removing an extra container from each deployed Pod and reducing the xDS update overhead by reducing an extra hop into the user space.

Proxying DNS requests to the server can aswell be achieved using eBPF but I'm not considering as part of this PoC.

## Potential Technology Stack:
K8S
Go
C
eBPF
TCP

## Additional Notes/links:
I already have a basic PoC that inject an eBPF program into the global cgroup and replace a connection destination IP with another from a hashmap controlled by a Go program using cgroup/connects hook.
I would like during the Hackathon to work on making this into a Consul in Kubernetes PoC as described above.

## Documentation

Documentation is in [here](docs/eBPF_solution.md)


