# Kubernetes Route Reflector (for MetalLB)

Route reflector server for use with [MetalLB BGP mode](https://metallb.universe.tf/concepts/bgp/).

Main functions:
- Keeps a stable connection to static upstream peers and distributes routes to them.
- Watches the node list of a Kubernetes cluster and peers with each one as a route reflection client. Each external and internal node address is setup as a passive peering connection.

## Usage

### Build

```sh
CGO_ENABLED=0 go build -o kube-route-reflector
```

### Binaries

You can find linux binaries with each release.

### Configuration

Example:

```yaml
clusters:
  - name: my-cluster
    host: "https://my-cluster:6443"
    # one or the other
    token: "service-acount-token"
    tokenFile: "my-service-account.token"

bgp:
  router_id: "10.0.1.1"
  local_as: 45678
  local_address: "10.0.128.0" # use this as the peer address in metalLB
  ipv4_multi_protocol: false # optional

  static_peers:
    - router_id: "10.0.2.1"
      peer_address: "10.0.128.1"
      peer_as: 45678

      # optional from here on
      auth_key: "password"
      passive: false
      route_reflector_client: false

    # more peers
```

### Run it

```sh
kube-route-reflector -config config.yaml -debug
```
