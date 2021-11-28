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

var ids map[string]bool

func initmaps() {
	temp := make(map[string]bool)
	ids = temp

	temp2 := make(map[string]*Model)
	models = temp2
}

func httpLoadModels(w http.ResponseWriter, r *http.Request) {

	modelsRes := []*ModelResponse{}

	//get file information on each model
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		log.Fatal(err)
	}

	//iterate and check size
	for _, modelFileInfo := range fileInfos {
		if modelFileInfo.Size() > int64(max_memory) {
			log.Print("file too large")
			respondWithError(w, http.StatusInternalServerError, "File too large.")
			return
		}

		//get unix timestamp
		stat_t := modelFileInfo.Sys().(*syscall.Stat_t)
		timestamp := timespecToTime(stat_t.Ctim)

		//get model name
		filename := strings.Split(modelFileInfo.Name(), ".")
		name := filename[0]
		//join
		uuid := name + "_" + strconv.Itoa(int(timestamp))

		//save in slice of requested models uuids
		ids[uuid] = true

		//if model isnt loaded proceed to load and add
		if _, ok := models[uuid]; !ok {
			path := filepath.Join(dir, modelFileInfo.Name())
			models[uuid], err = LoadModel(path)

			// models[uuid], err = predict.LoadModel(path)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, err.Error())
				log.Fatalln(err)
			}
		}
		//add data to http response
		modelsRes = append(modelsRes, &ModelResponse{
			Model:  name,
			SizeMB: float64(modelFileInfo.Size()) / 1000000.,
			UUID:   uuid,
		})
	}

	//loop over loaded models
	for id, _ := range models {
		//if any uuid is not among the requested values release from memory
		if _, ok := ids[id]; !ok {
			delete(models, id)
		}
	}

	err = respondWithJSON(w, http.StatusOK, modelsRes)
	if err != nil {
		log.Fatalln(err)
	}
}

func httpLoadModel(w http.ResponseWriter, r *http.Request) {

	//get para
	vars := mux.Vars(r)
	if vars["model"] == "" {
		return
	}
	filename := vars["model"] + ".cbm"

	path := filepath.Join(dir, filename)

	modelFileInfo, err := os.Stat(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		log.Fatal(err)
	}

	//check size
	if modelFileInfo.Size() > int64(max_memory) {
		respondWithError(w, http.StatusInternalServerError, "file too large")
		return
	}

	//make uuid
	stat_t := modelFileInfo.Sys().(*syscall.Stat_t)
	timestamp := timespecToTime(stat_t.Ctim)
	uuid := vars["model"] + "_" + strconv.Itoa(int(timestamp))

	//if there's no loaded model or its a different one
	if loadedModel == nil || uuid != loadedModel.UUID {
		//new model -- overload
		// loaded_model, err = predict.LoadModel(path)
		models[uuid], err = LoadModel(path)

		if err != nil {
			log.Fatalln(err)
		}
	}

	//prepare http response
	modelRes := &ModelResponse{
		Model:  vars["model"],
		SizeMB: float64(modelFileInfo.Size()) / 1000000.,
		UUID:   uuid,
	}
	err = respondWithJSON(w, http.StatusOK, modelRes)
	if err != nil {
		log.Fatal(err)
	}
}

func httpMakePrediction(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	if vars["model"] == "" {
		respondWithError(w, http.StatusBadRequest, "Missing model param.")
		log.Fatalln("Missing model param.")
		return
	}

	filename := vars["model"] + ".cbm"
	path := filepath.Join(dir, filename)

	modelFileInfo, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}

	if modelFileInfo.Size() > int64(max_memory) {
		respondWithError(w, http.StatusInternalServerError, "File too big.")
		return
	}

	model, err := LoadModel(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		log.Fatal(err)
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

	for i := 0; i < len(predArr); i++ {
		//launch goroutines
		go func(order int, my_chan chan<- []float64) {

			var res []float64
			pred, err := model.GetPrediction(predArr[order])

			if err != nil {
				log.Fatal(err)
			}
			//append order tag and result
			res = append(res, float64(order))
			res = append(res, pred)
			my_chan <- res

		}(i, resultsChan)
	}

	//init results list
	results := make([]float64, len(predArr))

	for i := 0; i < len(predArr); i++ {
		//assign results on indices based on tags
		val := <-resultsChan
		results[int(val[0])] = val[1]
	}
	//get uuid
	stat_t := modelFileInfo.Sys().(*syscall.Stat_t)
	timestamp := timespecToTime(stat_t.Ctim)
	uuid := vars["model"] + "_" + strconv.Itoa(int(timestamp))

	if vars["optional"] != "" {
		//case winner
		var winnerRes PredictWinnerResponse

		if vars["optional"] == "min" {
			winnerRes.Winner = findMinIndex(results)
		}
		if vars["optional"] == "max" {
			winnerRes.Winner = findMaxIndex(results)
		}

		winnerRes.ModelUUID = uuid
		winnerRes.Prediction = results[winnerRes.Winner]

		err = respondWithJSON(w, 200, winnerRes)
		if err != nil {
			log.Fatalln(err)
		}
		return
	}

	//if optional wasnt set
	predictResponse := &PredictResponse{}
	predictResponse.ModelUUID = uuid
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

func findMaxIndex(array []float64) int {
	var max float64 = array[0]

	//find max value
	for _, value := range array {
		if max < value {
			max = value
		}
	}
	//find index of max value
	for i, value := range array {
		if value == max {
			return i
		}
	}
	return 0
}

func findMinIndex(array []float64) int {
	var min float64 = array[0]

	//find max value
	for _, value := range array {
		if min > value {
			min = value
		}
	}
	//find index of max value
	for i, value := range array {
		if value == min {
			return i
		}
	}
	return 0
}

func timespecToTime(ts syscall.Timespec) int64 {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec)).Unix()
}
