#!/bin/sh
set -xe
echo ${1}
if ! $(command -v bpf2go >/dev/null 2>&1)
 then go install github.com/cilium/ebpf/cmd/bpf2go@master
fi
sudo pkill -9 nginx || true
${1} kill server || true && sudo $1 rm server || true
${1} run --name server -d nginx
serverIP="104.237.62.211" # ipconfig.me
sudo ./consul-ebpf-dataplane 169.1.1.1 ${serverIP} &
sleep 5 # wait for ebpf to be applied
${1} exec -it server curl -v -H 'Host: api.ipify.org' http://169.1.1.1/