### Run

Install [golangci-lint](https://golangci-lint.run/welcome/install/).

Update path to golangci-lint in Makefile if required.

Install [k6](https://grafana.com/docs/k6/latest/set-up/install-k6/)

Start
```shell
make all
```

Run load
```shell
k6 run load.js
```

### Stop

```shell
docker compose down -v
```