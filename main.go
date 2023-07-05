//go:build linux
// +build linux

package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strings"

	cebpf "github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

//go:generate bpf2go -cc clang -cflags "-O2 -g -Wall -Werror" bpf cgroup_connect4.c -- -I./headers

const bpfFSPath = "/sys/fs/bpf"

func main() {
	var err error
	// Name of the kernel function we're tracing
	fn := "count_sock4_connect"

	if len(os.Args) != 3 {
		log.Fatal("Not enough arguments supplied, need VIP and Backend Server IP's")
	}
	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	pinPath := path.Join(bpfFSPath, fn)
	if err := os.MkdirAll(pinPath, os.ModePerm); err != nil {
		log.Fatalf("failed to create bpf fs subpath: %+v", err)
	}

	log.Printf("Pin Path is %s", pinPath)

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, &cebpf.CollectionOptions{
		Maps: cebpf.MapOptions{
			// Pin the map to the BPF filesystem and configure the
			// library to automatically re-write it in the BPF
			// program so it can be re-used if it already exists or
			// create it if not
			PinPath: pinPath,
		},
	}); err != nil {
		log.Fatalf("loading objects: %v", err)
	}
	defer objs.Close()

	info, err := objs.bpfMaps.V4SvcMap.Info()
	if err != nil {
		log.Fatalf("Cannot get map info: %v", err)
	}
	log.Printf("Svc Map Info: %+v + str %s", info, objs.V4SvcMap.String())

	fakeVIP := net.ParseIP(os.Args[1])

	port := [2]byte{}

	binary.BigEndian.PutUint16(port[:], 443)

	fakeServiceKey := binary.LittleEndian.Uint32(fakeVIP.To4())

	fakeBackendIP := binary.LittleEndian.Uint32(net.ParseIP(os.Args[2]).To4())

	log.Printf("Loading with service 0x%x servicekey 0x%x", fakeBackendIP,
		fakeServiceKey)

	if err := objs.V4SvcMap.Update(fakeServiceKey, bpfConsulServers{fakeBackendIP}, cebpf.UpdateAny); err != nil {
		log.Fatalf("Failed Loading a fake service: %v", err)
	}

	// Get the first-mounted cgroupv2 path.
	cgroupPath, err := detectDockerCgroupPath()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Cgroup Path is %s", cgroupPath)

	// Link the proxy program to the default cgroup.
	l, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Attach:  cebpf.AttachCGroupInet4Connect,
		Program: objs.Sock4Connect,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	// log.Printf("returned service value: %+v", fakeServiceOut)
	log.Printf("Proxing for VIP %s to backend %s", os.Args[1], os.Args[2])
	ch := make(chan interface{})
	<-ch
}

// detectCgroupPath returns the first-found mount point of type cgroup2
// and stores it in the cgroupPath global variable.
func detectRootCgroupPath() (string, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// example fields: cgroup2 /sys/fs/cgroup/unified cgroup2 rw,nosuid,nodev,noexec,relatime 0 0
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) >= 3 && fields[2] == "cgroup2" {
			return fields[1], nil
		}
	}

	return "", errors.New("cgroup2 not mounted")
}

func detectDockerCgroupPath() (string, error) {
	const cgroupPath = "/sys/fs/cgroup/system.slice/"
	entries, err := os.ReadDir(cgroupPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "docker-") {
			return fmt.Sprintf("%s/%s", cgroupPath, e.Name()), nil
		}
	}
	return "", errors.New("no docker cgroup found")
}
