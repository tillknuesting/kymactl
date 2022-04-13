#!/bin/bash
for i in {1..100}
do
   echo "$i"
   cat <<EOF | kubectl apply -f -
apiVersion: inventory.kyma-project.io/v1alpha1
kind: Kyma
metadata:
  name: kyma-sample-$i
  namespace: pb
spec:
  components:
  - name: "istio"
    namespace: "istio-system"
  - name: "cluster-essentials"
  - name: "certificates"
    namespace: "istio-system"
  - name: "istio-resources"
  - name: "logging"
  - name: "tracing"
  - name: "kiali"
  - name: "monitoring"
  - name: "eventing"
  - name: "ory"
  - name: "api-gateway"
  - name: "service-catalog"
  - name: "service-catalog-addons"
  - name: "rafter"
  - name: "helm-broker"
  - name: "cluster-users"
  - name: "serverless"
  - name: "application-connector"
    namespace: "kyma-integration"
EOF
done