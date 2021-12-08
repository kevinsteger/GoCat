package main

import (
	"flag"
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

func main() {

	flag.IntVar(&port, "p", 8080, "the port to serve on. defaults to 8080")
	flag.IntVar(&max_channel, "b", 0, "maximum value of buffer channel. defaults to 0")
	flag.IntVar(&max_memory, "m", 64, "maximum size of models in megabytes. defaults to 64")
	flag.StringVar(&dir, "d", "../models/", "directory to load models from. defaults to parent dir/models")

	max_memory = max_memory * 1000000

	r := mux.NewRouter()
	r.HandleFunc("/models/load", httpLoadModels).Methods("GET")
	r.HandleFunc("/models/{model}/predict", httpMakePrediction).Methods("SEARCH")
	r.HandleFunc("/models/{model}/predict/{optional}", httpMakePrediction).Methods("SEARCH")

	http.Handle("/", r)

	fmt.Println("running on port: ", port)
	initmaps()
	apiLoadModels()

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), r))

}
