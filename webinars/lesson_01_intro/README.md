### Run

```shell
docker compose up -d
```

### Stop

```shell
docker compose down -v
```

### Graphite example

```shell
PORT=2003
SERVER=localhost

while true; do
	echo "my.example.metric.master.connections 90  `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.metric.slave.connections  200 `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.every_second.metric.slave.connections  200 `date +%s`" | nc ${SERVER} ${PORT}
	sleep 1
	echo "my.example.metric.master.connections 100  `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.metric.slave.connections  250 `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.every_second.metric.slave.connections  250 `date +%s`" | nc ${SERVER} ${PORT}
	sleep 1
	echo "my.example.metric.master.connections 110  `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.metric.slave.connections  350 `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.every_second.metric.slave.connections  350 `date +%s`" | nc ${SERVER} ${PORT}
	sleep 1
	echo "my.example.metric.master.connections 130  `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.metric.slave.connections  400 `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.every_second.metric.slave.connections  400 `date +%s`" | nc ${SERVER} ${PORT}
	sleep 1
	echo "my.example.metric.master.connections 105  `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.metric.slave.connections  200 `date +%s`" | nc ${SERVER} ${PORT}
	echo "my.example.every_second.metric.slave.connections  200 `date +%s`" | nc ${SERVER} ${PORT}
	sleep 1
	for i in `seq 1 30`; do
		echo "my.example.metric.master.connections 80  `date +%s`" | nc ${SERVER} ${PORT}
		echo "my.example.metric.slave.connections  100 `date +%s`" | nc ${SERVER} ${PORT}
		echo "my.example.every_second.metric.slave.connections  100 `date +%s`" | nc ${SERVER} ${PORT}
		sleep 1
	done
	date
done
```