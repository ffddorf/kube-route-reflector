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

    # insecure: set true when the cluster CA is self-signed
    insecure_disable_certificate_verify: false

api: # optional
  enabled: false # set true to enable api to get insights
  address: "localhost:6655" # default, optional

bgp:
  router_id: "10.0.1.1"
  local_as: 45678
  local_address: "10.0.128.0" # use this as the peer address in metalLB
  ipv4_multi_protocol: false # optional

  static_peers:
    - router_id: "10.0.2.1"
      peer_address: "10.0.128.1"
      peer_as: 45678

      # not required, defaults to false
      # set this to true for routers towards the edge
      # of your network, so they learn all routes
      # from clusters
      route_reflector_client: false

      # optional from here on
      auth_key: "password" # defaults to no auth
      passive: false # defaults to active

    # more peers
```

### Run it

```sh
kube-route-reflector -config config.yaml -debug
```

## Cluster Config

```
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-route-reflector
  namespace: kube-system
EOF

kubectl create clusterrole node-viewer --verb=get,list,watch --resource=nodes
kubectl create clusterrolebinding kube-route-reflector-binding --clusterrole=node-viewer --serviceaccount=kube-system/kube-route-reflector
```
