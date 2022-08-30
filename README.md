# GoCat
A highly scalable Catboost model server in Go

to build and run: 
```
go build -o main *.go && ./main
```

## environment variables:

**GOCAT_PORT** the local port to listen, default 8080

**GOCAT_MAX_CHANNEL** size of buffer channel, default 0 (no limit)

**GOCAT_MODEL_FOLDER** local folder to look for .cbm model files, default "../models/"

**GOCAT_MAX_MEMOR**Y max size in MB that all models loaded can occupy in memory, default 64

**GOCAT_CACHE_COUNT** LRU cache maximum items, default 100

**GOCAT_CACHE_TTL** time to live LRU cache value in seconds, default 10

## loading models:
```
curl http://localhost:8080/models/load
```

##make a prediction on the *addition* model:
```
curl --header "Content-Type: application/json" \
  --request SEARCH \
  --data '{ "features" : [[1,1],[2,3],[5,8],[13,21]] }' \
  http://localhost:8080/models/addition/predict
```
