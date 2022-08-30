# GoCat
A highly scalable Catboost model server in Go

### to build and run: 
```
go build -o main *.go && ./main
```

### environment variables:

**GOCAT_PORT** the local port to listen, default 8080

**GOCAT_MAX_CHANNEL** size of buffer channel, default 0 (no limit)

**GOCAT_MODEL_FOLDER** local folder to look for .cbm model files, default "../models/"

**GOCAT_MAX_MEMOR**Y max size in MB that all models loaded can occupy in memory, default 64

**GOCAT_CACHE_COUNT** LRU cache maximum items, default 100

**GOCAT_CACHE_TTL** time to live LRU cache value in seconds, default 10

### loading models:
```
curl http://localhost:8080/models/load
```
response:
```
[{
  "model": "addition",
  "uuid": "addition_1661879065",
  "sizeMB": 3.514784
}]
```

### make a prediction on the *addition* model:
```
curl --header "Content-Type: application/json" \
  --request SEARCH \
  --data '{ "features" : [[1,1],[2,3],[5,8],[13,21]] }' \
  http://localhost:8080/models/addition/predict
```
response:
```
{
"model_uuid": "addition_1661879065",
  "predictions": [
    2.5067490206808287,
    4.907823840927691,
    12.929173251603515,
    33.599922064959685
  ],
}
```
## optional predict parameters
Appending "max" or "min" to the end of the /predict endpoint will return only the winning prediction value and model index. e.g. /models/addition/predict/max results:
```
{
  "model_uuid": "addition_1661879065",
  "winner": 3,
  "prediction": 33.599922064959685
}
```
