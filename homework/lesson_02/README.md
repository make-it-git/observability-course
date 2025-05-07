### Run

```shell
docker compose up -d --build --force-recreate 
```

### Run simulated load
```shell
k6 run load.js
```

### For additional task
```shell
k6 run load-uneven-replicas.js
```

### Investigate
```
http://localhost:3000
```

### Stop

```shell
docker compose down -v
```