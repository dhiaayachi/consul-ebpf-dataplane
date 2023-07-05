## requirements
The requirements are:
- to be able to redirect a `TCP` connection from inside a container, originally to a VIP, to a Consul server IP. 
- The Consul server IP need to be injected by an external program into the eBPF program to allow load balancing between servers and replacing servers. 
- This need to be done on egress traffic and apply to cGroups
- ipv6 is out of the scope of the PoC
- DNS lookup proxying is our of the scope of this PoC

## eBPF hook

The identified eBPF hook that check those requirements is `cgroup/connect4`. This hook is called before creating a new TCP bind and provide access to rewrite the connection src/dst IPs as well as src/dst port, in our case only rewriting dst IP is needed.
```C
SEC("cgroup/connect4")
int sock4_connect(struct bpf_sock_addr *ctx)
```

[bpf_sock_add](https://elixir.bootlin.com/linux/latest/source/include/uapi/linux/bpf.h#L6435) is defined in `bpf.h` as follow:

```C
/* User bpf_sock_addr struct to access socket fields and sockaddr struct passed
 * by user and intended to be used by socket (e.g. to bind to, depends on
 * attach type).
 */
struct bpf_sock_addr {
	__u32 user_family;	/* Allows 4-byte read, but no write. */
	__u32 user_ip4;		/* Allows 1,2,4-byte read and 4-byte write.
				 * Stored in network byte order.
				 */
	__u32 user_ip6[4];	/* Allows 1,2,4,8-byte read and 4,8-byte write.
				 * Stored in network byte order.
				 */
	__u32 user_port;	/* Allows 1,2,4-byte read and 4-byte write.
				 * Stored in network byte order
				 */
	__u32 family;		/* Allows 4-byte read, but no write */
	__u32 type;		/* Allows 4-byte read, but no write */
	__u32 protocol;		/* Allows 4-byte read, but no write */
	__u32 msg_src_ip4;	/* Allows 1,2,4-byte read and 4-byte write.
				 * Stored in network byte order.
				 */
	__u32 msg_src_ip6[4];	/* Allows 1,2,4,8-byte read and 4,8-byte write.
				 * Stored in network byte order.
				 */
	__bpf_md_ptr(struct bpf_sock *, sk);
};
```
In our case the interesting filed is `user_ip4` which hold the address that the kernel will use to bind.

## cGroup to attach to

The idea here is to intercept the socket creation of each envoy container to bind to Consul xDS server and redirect that socket to the actual server address.  We have 2 options here:

- inject the eBPF program in the unified cGROUP v2, which will give us access to all the sockets created inside the host, we can then filter based on the destination IP (the magic VIP address and the destination port) and rewrite the destination only for those. The advantage of this approach is that all the rewrite are handled in a single eBPD program that is injected once at start time. The disadvantage is that it give access to every single socket in the box. To do so we need to attach the program to the following path `/sys/fs/cgroup/`
- inject the eBPF program in each envoy sidecar container cGroup, this would make it easier to manage the  C program as we will need less filetering logic but the eBPF program won't be central, which remove some possibilities, like being able to report to user space how many sidecars are connected to each server.

We can use[ Cilium Golang eBPF library](https://github.com/cilium/ebpf) to inject the eBPF program using go, for example from the consul-k8s controller at boot (if we go with option 1)

## injecting Consul Servers into the eBPF program

Communication between user-space and an eBPF program could be done using a HashMap or a RingBuffer. In our use case hashmap make more sense, we need to have for each virtual IP assigned to an envoy sidecar an entry in the HashMap that point to the IP of the server that should be used for that sidecar.
The eBPF program will lookup the HashMap using:

```C
/*
 * bpf_map_lookup_elem
 *
 * 	Perform a lookup in *map* for an entry associated to *key*.
 *
 * Returns
 * 	Map value associated to *key*, or **NULL** if no entry was
 * 	found.
 */
static void *(*bpf_map_lookup_elem)(void *map, const void *key) 
```

Injecting the value in the HashMap from userspace, could be done using Golang and  [Cilium Golang eBPF library](https://github.com/cilium/ebpf). Using bpf2go we can generate binding to the Map in go and use go to inject the servers values.