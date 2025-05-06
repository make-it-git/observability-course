### Run

```shell
docker compose up -d --build --force-recreate 
```

### Run simulated load
```shell
k6 run load.js
```

### Run single request to see outcome
```shell
 curl -X POST \
  http://localhost:8080/request-ride \
  -H 'Content-Type: application/json' \
  -d '{
    "pickup": "locationA",
    "dropoff": "locationB"
  }'

```

### Investigate
```
http://localhost:3000
```

### Stop

```shell
docker compose down -v
```