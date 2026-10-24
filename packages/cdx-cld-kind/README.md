# cdx-cld-kind

An sbx mixin for the built-in `codex` agent (the base). It adds Claude Code,
starts a single-node kind (Kubernetes-in-Docker) cluster on boot, and deploys the
Patronus harness for both agents with the patronus CLI, pinned to v2.7.0
through the repo installer (`install.sh --version`).

## What it installs

| Piece | Mechanism |
|-------|-----------|
| Claude Code CLI | `npm -g`; OAuth `/login` inside the VM |
| patronus v2.7.0 | `install.sh --version v2.7.0` (checksum-verified) |
| Codex harness | `--profile core-profile-cx`, plus `ticket` (tk) |
| Claude harness | `--profile core` (includes tk) |
| kind v0.33.0, kubectl v1.37.1 | `kind-install.sh`, in `~/.local/bin` |
| kind cluster | `kind-ensure.sh`, in the background on every start |

## Run it

kind and the harness need memory, which is a create-time flag:

```sh
sbx create codex --kit ~/.patronus/packages/cdx-cld-kind/ --name cdx-cld-kind --cpus 4 --memory 9g
sbx run --name cdx-cld-kind
```

The cluster takes 1-2 minutes on first boot (see `~/.kind-boot.log`). Then:

```sh
kubectl get nodes
kubectl get pods -A
```

## How kind runs in the microVM

A stock `kind create cluster` fails in the sandbox for two reasons. A root
startup step fixes both on every boot, because both reset on stop/start:

```
  blocker                               fix (kind-prep.sh, root)
  ------------------------------------  ------------------------------------
  inotify max_user_instances = 128      sysctl: instances 8192,
    -> kubeadm bootstrap times out        watches 524288
  no /dev/kmsg in the node              mknod /dev/kmsg c 1 11
    -> kubelet cannot start               (privileged nodes inherit it)
```

`kind-ensure.sh` waits for Docker, creates the cluster if it is missing, and
recreates it when its node container is not running.

## Caveats

- The kind node has no working registry egress, and `kind load docker-image`
  fails against the sandbox's Docker image layout. Workloads must use images
  preloaded in `kindest/node`, or a registry mirror configured in the node's
  containerd.
- Egress is allow-all (`**`). Real keys stay on the host, but agent-authored
  code can reach any host. Replace `**` with a host allowlist to restore the
  boundary.
- To skip the cluster, remove the two `setup.startup` steps or run
  `kind delete cluster`. Set `KIND_CLUSTER_NAME` to rename it.
