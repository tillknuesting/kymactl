#!/bin/bash
kubectl create ns pb
FROM=202
TO=202
i=$FROM
while [[ $i -le $TO ]]
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
  ((i = i + 1))
done
SECONDS=0
while [[ $(kubectl get kyma kyma-sample-${TO} -n pb -o 'jsonpath={..status.status}') != "success" ]]; do echo "Waiting for $(kubectl get kyma kyma-sample-${TO} -n pb -o 'jsonpath={..status.waitingFor}')"; sleep 2; done

echo "Last component reconciled in $SECONDS sec."

if [ $SECONDS -ge 100 ]
then
  echo "Reconciliation took too long. Expected time: 68 - 90 seconds."
  exit 1
fi