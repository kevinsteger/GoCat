# GoCat
Catboost model server in Go

to build and run: 
```
go build -o main *.go && ./main
```

loading models:
```
curl http://localhost:8080/models/load
```

make a prediction:
```
curl --header "Content-Type: application/json" \
  --request SEARCH \
  --data '{ "features" : [[1,1],[2,3],[5,8],[13,21]] }' \
  http://localhost:8080/models/addition/predict
```
