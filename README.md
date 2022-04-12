# Project setup

The project was created with [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder):

```
kubebuilder init --domain kyma-project.io --repo github.com/kyma-incubator/kymactl
kubebuilder create api --group inventory --version v1alpha1 --kind Cluster
kubebuilder create api --group inventory --version v1alpha1 --kind HelmComponent
kubebuilder create api --group inventory --version v1alpha1 --kind Kyma
```

# Testing

## Locally

Start controller:
```
make manifests
make install 
make run
```

## In the cluster

You need a write access to some docker registry. You can use github (ghcr.io). Just create personal access token (developer settings) and use it as a password to login:
```
docker login ghcr.io -u your_github_user 
```
Build and push controller:
```
make docker-build docker-push IMG=ghcr.io/pbochynski/kyma-operator:0.0.1
```

Run controller:
```
make deploy IMG=ghcr.io/pbochynski/kyma-operator:0.0.1
```

## Generate sample data

```
kubectl create ns pb
./config/samples/sample-data.sh
```
