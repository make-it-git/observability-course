```shell
python3 -m venv myenv; source myenv/bin/activate
pip install -r requirements.txt
```

```shell
docker compose up -d
```

### Generate sinusoidal value
```shell
python generate.py
```

### After a few minutes of generation run prediction
```shell
python predict.py
```