#!/usr/bin/env sh
# VM prep for kind, as root on every start (this state resets on stop/start):
#   - raise inotify limits, or kubeadm bootstrap times out
#   - create /dev/kmsg, or kubelet fails; privileged kind nodes inherit it
sysctl -w fs.inotify.max_user_instances=8192 fs.inotify.max_user_watches=524288 >/dev/null 2>&1 || true
[ -e /dev/kmsg ] || mknod /dev/kmsg c 1 11 || true
