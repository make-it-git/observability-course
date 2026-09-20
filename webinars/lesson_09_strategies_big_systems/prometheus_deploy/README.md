### Install

```shell
minikube start

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

helm search repo prometheus-community/kube-prometheus-stack --versions

helm install prometheus prometheus-community/kube-prometheus-stack --version 70.0.1 --namespace monitoring --create-namespace

kubectl get pods -n monitoring
```

### Components

prometheus operator
node exporter (daemonset)
// kubectl -n monitoring describe daemonsets.apps prometheus-prometheus-node-exporter
kube state metrics (metrics from API server)
// kubectl get clusterrolebindings.rbac.authorization.k8s.io prometheus-kube-state-metrics -o yaml
// kubectl get clusterrole prometheus-kube-state-metrics -o yaml

grafana

### Access

```shell
kubectl port-forward svc/prometheus-operated -n monitoring 9090:9090
kubectl port-forward svc/prometheus-grafana -n monitoring 3000:80
kubectl get secret -n monitoring prometheus-grafana -o jsonpath="{.data.admin-password}" | base64 --decode
```

Sample metrics

```
kube_pod_container_status_running
node_load1
```

### Install sample app

```shell
eval $(minikube docker-env)
docker build -t my-go-app:latest .
kubectl apply -f k8s/service.yaml -f k8s/deployment.yaml
kubectl port-forward svc/my-service 8080:8080
curl localhost:8080
curl localhost:8080/metrics | grep my_go_app
```

### Set monitoring

First check http://localhost:9090/targets and http://localhost:9090/rules

```shell
kubectl get servicemonitors.monitoring.coreos.com -A
kubectl get prometheusrules.monitoring.coreos.com -A
```

Then apply

```shell
kubectl apply -f k8s/service-monitor.yaml -f k8s/prometheus-rule.yaml
```