package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

type ModelResponse struct {
	Model  string  `json:"model"`
	UUID   string  `json:"uuid"`
	SizeMB float64 `json:"sizeMB"`
}

type SamplesRequest struct {
	InputArr [][]interface{} `json:"inputArr"`
}

type PredictResponse struct {
	ModelUUID   string    `json:"model_uuid"`
	Predictions []float64 `json:"predictions"`
}

type PredictWinnerResponse struct {
	ModelUUID  string  `json:"model_uuid"`
	Winner     int     `json:"winner"`
	Prediction float64 `json:"prediction"`
}

var models map[string]*Model
var loadedModel *Model

var requestedModels map[string]bool

func initmaps() {
	temp := make(map[string]bool)
	requestedModels = temp

	temp2 := make(map[string]*Model)
	models = temp2
}

func httpLoadModels(w http.ResponseWriter, r *http.Request) {

	modelsRes := []*ModelResponse{}

	//check total size before loading
	size, err := dirSize(dir)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		log.Fatal(err)
	}

	if size > int64(max_memory) {
		log.Print("Total memory for models exceeds configured limit of" + strconv.Itoa(max_memory) + "(MB)")
		respondWithError(w, http.StatusForbidden, "Total memory for models exceeds configured limit of"+strconv.Itoa(max_memory)+"(MB)")
		return
	}

	//get file information on each model
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		log.Fatal(err)
	}

	//iterate and check size
	for _, modelFileInfo := range fileInfos {

		//get unix timestamp
		stat_t := modelFileInfo.Sys().(*syscall.Stat_t)
		timestamp := timespecToTime(stat_t.Ctim)

		//get model name
		filename := strings.Split(modelFileInfo.Name(), ".")
		name := filename[0]
		//join
		uuid := name + "_" + strconv.Itoa(int(timestamp))

		//save in slice of requested models names
		requestedModels[name] = true

		//if model isnt loaded proceed to load and add
		//or if it's an old version -- overload
		_, ok := models[name]
		if !ok || models[name].UUID != uuid {
			path := filepath.Join(dir, modelFileInfo.Name())
			models[name], err = LoadModel(path)

			if err != nil {
				respondWithError(w, http.StatusInternalServerError, err.Error())
				log.Fatalln(err)
			}
			models[name].UUID = uuid
		}
		//add data to http response
		modelsRes = append(modelsRes, &ModelResponse{
			Model:  name,
			SizeMB: float64(modelFileInfo.Size()) / 1000000.,
			UUID:   uuid,
		})
	}

	//loop over loaded models
	for name, _ := range models {
		//if any model name is not among the requested values release from memory
		if _, ok := requestedModels[name]; !ok {
			delete(models, name)
		}
	}

	err = respondWithJSON(w, http.StatusOK, modelsRes)
	if err != nil {
		log.Fatalln(err)
	}
}

func httpMakePrediction(w http.ResponseWriter, r *http.Request) {
	var err error
	vars := mux.Vars(r)

	if vars["model"] == "" {
		respondWithError(w, http.StatusBadRequest, "Missing model param.")
		log.Println("Missing model param.")
		return
	}

	if models[vars["model"]] == nil {
		respondWithError(w, http.StatusInternalServerError, "Model not loaded in memory.")
		return
	}

	resultsChan := make(chan []float64, max_channel)

	//decode request
	var req SamplesRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		log.Fatalln(err)
	}

	defer r.Body.Close()
	defer close(resultsChan)

	predArr := req.InputArr
	results := make([]float64, len(predArr))
	var index int   //index of extreme value
	var ext float64 //min or max

	for i := 0; i < len(predArr); i++ {
		//launch goroutines
		go func(order int, my_chan chan<- []float64) {

			var res []float64
			pred, err := models[vars["model"]].GetPrediction(predArr[order])

			if err != nil {
				respondWithError(w, http.StatusInternalServerError, err.Error())
				log.Fatal(err)
			}
			//append order tag and result
			res = append(res, float64(order))
			res = append(res, pred)
			my_chan <- res

		}(i, resultsChan)

		val := <-resultsChan

		if i == 0 {
			ext = val[1]
		}
		if vars["optional"] == "max" {
			if val[1] > ext {
				ext = val[1]
				index = i
			}
		} else if vars["optional"] == "min" {
			if val[1] < ext {
				ext = val[1]
				index = i
			}
		} else {
			results[int(val[0])] = val[1]
		}
	}

	if vars["optional"] != "" {
		var winnerRes PredictWinnerResponse

		winnerRes.Winner = index
		winnerRes.ModelUUID = models[vars["model"]].UUID
		winnerRes.Prediction = ext

		err = respondWithJSON(w, 200, winnerRes)
		if err != nil {
			log.Fatalln(err)
		}
		return
	}

	//if optional wasnt set
	predictResponse := &PredictResponse{}
	predictResponse.ModelUUID = models[vars["model"]].UUID
	predictResponse.Predictions = results

	err = respondWithJSON(w, 200, predictResponse)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		log.Fatalln(err)
	}
}

// helper functions

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)

	return nil
}

func timespecToTime(ts syscall.Timespec) int64 {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec)).Unix()
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
