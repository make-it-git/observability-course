```shell

# Start cluster
minikube start --memory=8192 --cpus=4

# Install istio
curl -sL https://istio.io/downloadIstioctl | sh -
~/.istioctl/bin/istioctl x precheck
~/.istioctl/bin/istioctl install --set profile=demo -y

# Install prometheus & kiali
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.26/samples/addons/prometheus.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.26/samples/addons/kiali.yaml

# Verify proxy statuses
~/.istioctl/bin/istioctl proxy-status

# Sample app
kubectl create namespace app
kubectl create namespace test
kubectl label namespace app istio-injection=enabled
kubectl label namespace test istio-injection=enabled

kubectl apply -n app -f hello.yaml
kubectl -n app get pod

# Verify access
kubectl run -n test curl --image=curlimages/curl --restart=Never -- /bin/sh -c 'sleep 3600'
kubectl -n test exec -it curl -- curl -v hello-world.app.svc.cluster.local

# Verify proxy statuses again
~/.istioctl/bin/istioctl proxy-status

# View traffic
kubectl -n istio-system port-forward svc/kiali 20001:20001
```

