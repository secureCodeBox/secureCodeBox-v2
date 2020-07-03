# secureCodeBox - Kubernetes Auto-Discovery

Automatically finds security relevant resources in the kubernetes API and dispatches matching secureCodeBox Scans against them.

The Kubernetes Auto-Discovery needs to be deployed along side the secureCodeBox Operator. The Auto-Discovery services discovers targets, the Operator executes the scans against them.

## Installation

> **Note** Installation is still todo

```bash

# Until it's officially release you can only run the auto discovery service locally against your current kubectl context
cd ./auto-discovery/kubernetes/
make run

# Coming soon helm installation
# helm install -n securecodebox-system k8s-auto-discovery ./auto-discovery/kubernetes/
```

## Enabling Namespaces for Auto-Discovery

The Auto-Discovery only creates Scans for namespaces which are explicitly enabled.
You can enable a namespace by annotating it

> **Note** The namespaces you are using need to have the `trivy`, `kube-hunter`, `zap`, `nikto` and `sslyze` ScanTypes installed to work correctly. See the top level readme for install instructions

```bash
kubectl annotate namespaces juice-shop auto-discovery.experimental.securecodebox.io/enabled="true"
```
