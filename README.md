# Overlay Network Test

Test Overlay Network of your Kubernetes Cluster

This is a [Go client](https://github.com/kubernetes/client-go) of the [Overlay Network Test](https://github.com/mcsps/use-cases/blob/master/README.md#k8s-overlay-network-test), a shell script paired with a DaemonSet to check connectivity in Overlay Network in Kubernetes Cluster.

## Requirements

* kube-config with connection to a working Kubernetes Cluster
* access `kube-system` namespace where the application will deploy (needs priviledged mode for network ping command)

## Usage

Download artifact from [Release Page](https://github.com/eumel8/overlaytest/releases) and execute.

## Docker Image

* [source](https://github.com/mcsps/swiss-army-knife/tree/mcsps)

* [repo](https://mtr.devops.telekom.de/repository/mcsps/swiss-army-knife?tab=tags)


## Credits

Frank Kloeker f.kloeker@telekom.de

Life is for sharing. If you have an issue with the code or want to improve it, feel free to open an issue or an pull request.
