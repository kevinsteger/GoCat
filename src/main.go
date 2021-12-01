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
	r.HandleFunc("/models/load", httpLoadModels)
	r.HandleFunc("/models/{model}/load", httpLoadModel)
	r.HandleFunc("/models/{model}/predict", httpMakePrediction)
	r.HandleFunc("/models/{model}/predict/{optional}", httpMakePrediction)

	http.Handle("/", r)

	fmt.Println("running on port: ", port)
	initmaps()

	size, err := dirSize(dir)
	if err != nil {
		log.Fatal(err)
	}
	if size > int64(max_memory) {
		log.Fatal("Total memory for models exceeds configured limit of" + strconv.Itoa(max_memory) + "(MB)")
	}

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), r))

}
