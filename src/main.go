package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	lru "github.com/hnlq715/golang-lru"
)

var port int
var max_channel int
var max_memory int
var dir string
var cache_count int
var cache_ttl int
var arc *lru.ARCCache

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	} else {
		return value
	}
}

func main() {

	port, _ = strconv.Atoi(getenv("GOCAT_PORT", "8080"))
	max_channel, _ = strconv.Atoi(getenv("GOCAT_MAX_CHANNEL", "0"))
	dir = getenv("GOCAT_MODEL_FOLDER", "../models/")
	max_memory, _ = strconv.Atoi(getenv("GOCAT_MAX_MEMORY", "64"))
	cache_count, _ = strconv.Atoi(getenv("GOCAT_CACHE_COUNT", "100"))
	cache_ttl, _ = strconv.Atoi(getenv("GOCAT_CACHE_TTL", "10"))

	max_memory = max_memory * 1000000

	var err error
	arc, err = lru.NewARCWithExpire(cache_count, time.Duration(cache_ttl)*time.Second)
	if err != nil {
		log.Println("cannot initialize arc: ", err)
	} else {
		defer arc.Purge()
	}

	r := mux.NewRouter()
	r.HandleFunc("/models/load", httpLoadModels).Methods("GET")
	r.HandleFunc("/models/{model}/predict", httpMakePrediction).Methods("SEARCH")
	r.HandleFunc("/models/{model}/predict/{optional}", httpMakePrediction).Methods("SEARCH")

	http.Handle("/", r)

	fmt.Println("running on port: ", port)
	fmt.Println("maximum channels: ", max_channel)
	fmt.Println("model directory: ", dir)
	fmt.Println("max memory: ", max_memory)
	fmt.Println("cache count: ", cache_count)
	fmt.Println("cache ttl: ", cache_ttl)

	initmaps()
	apiLoadModels()

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), r))

}
