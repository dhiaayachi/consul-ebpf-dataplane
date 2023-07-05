# The development version of clang is distributed as the 'clang' binary,
# while stable/released versions have a version number attached.
# Pin the default clang to a stable version.
CLANG ?= clang-14
STRIP ?= llvm-strip-14
CFLAGS := -O2 -g -Wall -Werror $(CFLAGS)

# Obtain an absolute path to the directory of the Makefile.
# Assume the Makefile is in the root of the repository.
REPODIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
UIDGID := $(shell stat -c '%u:%g' ${REPODIR})

# Prefer podman if installed, otherwise use docker.
# Note: Setting the var at runtime will always override.
CONTAINER_ENGINE ?= $(if $(shell command -v podman), podman, docker)
CONTAINER_RUN_ARGS ?= $(if $(filter ${CONTAINER_ENGINE}, podman),, --user "${UIDGID}")
export ${CONTAINER_ENGINE}


.PHONY: #all clean generate run

.DEFAULT_TARGET = run

clean:
	-$(RM) *.o
	-$(RM) bpf_bpfel.*
	-$(RM) consul-ebpf-dataplane

format:
	find . -type f -name "*.c" | xargs clang-format -i

build: generate
	go build

# $BPF_CLANG is used in go:generate invocations.
generate: export BPF_CLANG := $(CLANG)
generate: export BPF_CFLAGS := $(CFLAGS)
generate:
	go generate ./
	
run: build
	./run.sh ${CONTAINER_ENGINE} 
