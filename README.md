# Project setup

```
kubebuilder init --domain kyma-project.io --repo github.com/kyma-incubator/kymactl
kubebuilder create api --group inventory --version v1alpha1 --kind Cluster
kubebuilder create api --group inventory --version v1alpha1 --kind HelmComponent
kubebuilder create api --group inventory --version v1alpha1 --kind Kyma


```