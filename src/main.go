package main

import (
    "os"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

var port int
var max_channel int
var max_memory int
var dir string

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
        
	max_memory = max_memory * 1000000

	r := mux.NewRouter()
	r.HandleFunc("/models/load", httpLoadModels).Methods("GET")
	r.HandleFunc("/models/{model}/predict", httpMakePrediction).Methods("SEARCH")
	r.HandleFunc("/models/{model}/predict/{optional}", httpMakePrediction).Methods("SEARCH")

	http.Handle("/", r)

	fmt.Println("running on port: ", port)
    fmt.Println("maximum channels: ", max_channel)
    fmt.Println("model directory: ", dir)
    fmt.Println("max memory: ", max_memory)
	
    initmaps()
	apiLoadModels()

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), r))

}
