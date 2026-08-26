```shell
brew install go-jsonnet
git clone https://github.com/grafana/grafonnet-lib.git grafonnet
jsonnet -J grafonnet dashboard.jsonnet > ./docker/grafana/provisioning/dashboards/dashboard.json
docker compose up -d
```
