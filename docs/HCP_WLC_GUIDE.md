# HCP & WLC Guide (LXD on MAAS)

This guide explains how to use the two LXD-based features of the CAPMAAS provider:

- **HCP — Host Control Plane**: a cluster whose **bare-metal** MAAS machines are
turned into **LXD VM hosts**. These hosts provide the capacity that workload
clusters run on.
- **WLC — Workload Cluster**: a cluster whose machines are dynamically created as **LXD VMs**
on the HCP hosts (instead of on bare metal).

The typical flow is: **deploy an HCP cluster first**, let its bare-metal nodes
register as LXD hosts in MAAS, then **deploy one or more WLC clusters** whose VMs
are scheduled onto those hosts.

```
            HCP cluster (bare metal)                 WLC cluster (LXD VMs)
   ┌──────────────────────────────────────┐   ┌──────────────────────────────┐
   │  CP node (BM)  ─ LXD host             │   │  CP VM   ─┐                   │
   │  worker (BM)   ─ LXD host  ◀──────────┼───┼─ worker VM┴─ run as LXD VMs   │
   │  worker (BM)   ─ LXD host             │   │             on the HCP hosts  │
   └──────────────────────────────────────┘   └──────────────────────────────┘
        lxdConfig.enabled: true                    spec.lxd.enabled: true
        --cluster-role=hcp                          --cluster-role=wlc
```

---

## 1. Network prerequisites (read this first)

A bare-metal machine can be turned into an LXD host **only** if its **PXE boot
interface** has one of these two simple topologies:


| Supported | PXE boot interface looks like                         | What the initializer does          |
| --------- | ----------------------------------------------------- | ---------------------------------- |
| ✅         | **Physical interface only**, no bridge on the PXE NIC | Creates the LXD bridge on that NIC |
| ✅         | **A bridge already exists** on the PXE boot interface | Reuses the existing bridge         |


**Not supported (yet):** any complex topology on the PXE path — **bond**,
**bridge-on-bond**, **VLAN**, or combinations of these. If the PXE boot interface
is part of a bond/VLAN stack, the host will not initialize correctly.

> Tip: check the machine's interfaces in the MAAS UI. The PXE interface is the
> one with the **PXE** checkmark. It must be either a plain *Physical* NIC or a
> *Bridge* whose only member is a single *Bridged physical* NIC.

---

## 2. The `--cluster-role` flag

The controller enables extra controllers based on the role you run it with:


| `--cluster-role`      | Use for               | Extra controller enabled              |
| --------------------- | --------------------- | ------------------------------------- |
| `standard` (or empty) | normal MAAS clusters  | none (base infra only)                |
| `hcp`                 | HCP host clusters     | **HMC** — Host Maintenance Controller |
| `wlc`                 | LXD workload clusters | **VEC** — VM Evacuation Controller    |


Set it on the controller-manager Deployment, e.g.:

```yaml
args:
  - --cluster-role=hcp   # or wlc
```

---

## 3. Deploy an HCP cluster

Template: `[templates/exp/hcp-cluster-template.yaml](../templates/exp/hcp-cluster-template.yaml)`

The `MaasCluster` enables LXD on the whole cluster, so every machine it
provisions is registered as an LXD VM host:

```yaml
kind: MaasCluster
spec:
  dnsDomain: ${DNS_DOMAIN}
  failureDomains:
    - default
  lxdConfig:
    enabled: true            # register this cluster's machines as LXD hosts
    storageBackend: zfs      # zfs | dir
    storageSize: "50"        # LXD storage pool size in GB
    skipNetworkUpdate: true  # don't mutate existing host networks
```

Control-plane and worker machines have **no `spec.lxd` block** — they run on bare
metal and are tagged `lxd-host`.

Render and apply (controller must run with `--cluster-role=hcp`):

```bash
export CLUSTER_NAME=hcp
export DNS_DOMAIN=maas
export KUBERNETES_VERSION=v1.33.5
export MACHINE_IMAGE=custom/u-2204-0-k-1335-0
export CONTROL_PLANE_MACHINE_COUNT=1
export WORKER_MACHINE_COUNT=2
export CONTROL_PLANE_ZONE=default
export WORKER_ZONE=default
export LXD_RESOURCE_POOL=lxd-hosts   # use a dedicated pool, not "default" (see note below)
export LXD_STORAGE_SIZE=50
export CONTROL_PLANE_MACHINE_MINCPU=4
export CONTROL_PLANE_MACHINE_MINMEMORY=8192
export CONTROL_PLANE_MACHINE_MINSTORAGE=60
export WORKER_MACHINE_MINCPU=4
export WORKER_MACHINE_MINMEMORY=8192
export WORKER_MACHINE_MINSTORAGE=60

envsubst < templates/exp/hcp-cluster-template.yaml | kubectl apply -f -
```

**Verify**: once the nodes are provisioned, open the MAAS UI → **KVM / LXD** and
confirm each HCP node is registered as an LXD host.

> **Use a dedicated resource pool — do not use `default`.** Create a separate MAAS
> resource pool (e.g. `lxd-hosts`) and place only the bare-metal machines you want
> to become LXD hosts in it. The `default` pool typically contains every machine
> MAAS knows about, so using it risks pulling unrelated machines into the cluster.
> A dedicated pool keeps HCP host selection predictable and isolated.

---

## 4. Deploy a WLC cluster

Template: `[templates/exp/lxd-cluster-template.yaml](../templates/exp/lxd-cluster-template.yaml)`

Here individual machines opt into being LXD VMs through the per-machine
`spec.lxd` block on the `MaasMachineTemplate`:

```yaml
kind: MaasMachineTemplate
spec:
  template:
    spec:
      image: ${MACHINE_IMAGE}
      minCPU: ${CONTROL_PLANE_MACHINE_MINCPU}
      minMemory: ${CONTROL_PLANE_MACHINE_MINMEMORY}
      resourcePool: ${LXD_RESOURCE_POOL}
      lxd:
        enabled: true              # create this machine as an LXD VM on an HCP host
        vmConfig:
          diskSize: ${CONTROL_PLANE_MACHINE_MINSTORAGE}
          autoStart: true
```

Machines **without** a `spec.lxd` block (e.g. the worker template in this example)
are still provisioned on bare metal — you can mix VM and bare-metal pools in one
WLC.

> **Recommendation: run worker nodes on bare metal, not as LXD VMs.**

Render and apply (controller must run with `--cluster-role=wlc`):

```bash
export CLUSTER_NAME=wlc
export DNS_DOMAIN=maas
export KUBERNETES_VERSION=v1.33.5
export MACHINE_IMAGE=custom/u-2204-0-k-1335-0
export CONTROL_PLANE_MACHINE_COUNT=1
export WORKER_MACHINE_COUNT=2
export LXD_ZONE=default
export WORKER_ZONE=default
export LXD_RESOURCE_POOL=lxd-hosts   # dedicated pool, not "default" (see note below)
export CONTROL_PLANE_MACHINE_MINCPU=4
export CONTROL_PLANE_MACHINE_MINMEMORY=8192
export CONTROL_PLANE_MACHINE_MINSTORAGE=60
export WORKER_MACHINE_MINCPU=4
export WORKER_MACHINE_MINMEMORY=8192
export WORKER_MACHINE_IMAGE=custom/u-2204-0-k-1335-0
export WORKER_MACHINE_RESOURCEPOOL=lxd-hosts   # dedicated pool, not "default"
export WORKER_MACHINE_TAG=worker

envsubst < templates/exp/lxd-cluster-template.yaml | kubectl apply -f -
```

> **Use a dedicated resource pool — do not use `default`.** As with HCP, point
> `LXD_RESOURCE_POOL` (and any bare-metal worker pool) at a dedicated MAAS resource
> pool rather than `default`, so machine selection stays scoped to the hardware you
> intend to use.

---

## 5. Key fields reference

`**MaasCluster.spec.lxdConfig`** (HCP):


| Field               | Default | Description                                      |
| ------------------- | ------- | ------------------------------------------------ |
| `enabled`           | `false` | Register this cluster's machines as LXD VM hosts |
| `storageBackend`    | `zfs`   | LXD storage backend (`zfs`, `dir`, …)            |
| `storageSize`       | `"50"`  | Storage pool size in GB                          |
| `skipNetworkUpdate` | `true`  | Skip mutating existing host networks             |


`**MaasMachine(Template).spec.lxd**` (WLC):


| Field                  | Default | Description                                     |
| ---------------------- | ------- | ----------------------------------------------- |
| `enabled`              | `false` | Create this machine as an LXD VM on an HCP host |
| `vmConfig.diskSize`    | `60`    | VM disk size in GB                              |
| `vmConfig.`storagePool | —       | LXD storage pool to use                         |
| `vmConfig.autoStart`   | `true`  | Auto-start the VM                               |


---

## 6. Troubleshooting

- **Host never becomes an LXD host** — check the PXE boot interface topology
(see [§1](#1-network-prerequisites-read-this-first)); bond/VLAN PXE paths are
not supported.
- **VM creation fails** — confirm the HCP cluster is healthy and its nodes appear
as LXD hosts in MAAS, and that the host has free CPU/memory/storage.
- **Wrong controllers running** — make sure the controller for each cluster is
started with the correct `--cluster-role` (`hcp` for HCP, `wlc` for WLC).

