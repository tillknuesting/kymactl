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
make install 
make deploy IMG=ghcr.io/pbochynski/kyma-operator:0.0.1
```

## Generate sample data

```
kubectl create ns pb
./config/samples/sample-data.sh
```

# Performance

## Kubernetes client rate limiting

Default kubernetes client is configured with QPS=20 and Burst=30. For the scenario where 1 Kyma resource creates 18 modules we can create/update at most une Kyma installation per second. More reasonable setting would be:

```
cfg.QPS = 100
cfg.Burst = 100
```

This is inspired by prometheus operator settings:
https://github.com/prometheus-operator/prometheus-operator/blob/bbf82e22de6fa2bf3f28cecdc82a8695748d5017/pkg/k8sutil/k8sutil.go#L96-L97

See also: https://github.com/voyagermesh/voyager/issues/640

## Reconciliation queue rate limiting

The default rate limiting was changed to allow more reconciliations (bigger bucket) and increase the time of first retry from 5ms to 1s:

```
func CustomRateLimiter() ratelimiter.RateLimiter {
	return workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, 1000*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(30), 200)})
}
```