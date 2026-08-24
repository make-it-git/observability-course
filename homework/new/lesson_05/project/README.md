### Run

```shell
docker compose up -d --build --force-recreate 
```

### Run simulated load
```shell
k6 run load.js
```

### Investigate
```
http://localhost:3000
```

### Stop

```shell
docker compose down -v
```