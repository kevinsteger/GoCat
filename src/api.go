package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var models map[string]*Model

var requestedModels map[string]bool

func initmaps() {
	temp := make(map[string]bool)
	requestedModels = temp

	temp2 := make(map[string]*Model)
	models = temp2
}

func apiLoadModels() (modelsRes []*ModelResponse, code int, err error) {
	//check total size before loading
	size, err := dirSize(dir)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if size > int64(max_memory) {
		return nil, http.StatusForbidden, errors.New("Total memory for models exceeds configured limit of" + strconv.Itoa(max_memory) + "(MB)")
	}

	//get file information on each model
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, http.StatusInternalServerError, err
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
				return nil, http.StatusInternalServerError, err
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

	return modelsRes, http.StatusOK, err
}

func apiMakePrediction(req *SamplesRequest, model string, optional string) (interface{}, error) {
	resultsChan := make(chan []float64, max_channel)
	errorChan := make(chan error, 1)

	defer close(resultsChan)
	defer close(errorChan)

	features := req.Features
	results := make([]float64, len(features))
	var index int   //index of extreme value
	var ext float64 //min or max

	for i := 0; i < len(features); i++ {
		//launch goroutines
		go func(order int, my_chan chan<- []float64, err_chan chan<- error) {

			var res []float64
			pred, err := models[model].GetPrediction(features[order])

			if err != nil {
				err_chan <- err
			}
			//append order tag and result
			res = append(res, float64(order))
			res = append(res, pred)
			my_chan <- res
			err_chan <- nil

		}(i, resultsChan, errorChan)

		val := <-resultsChan
		err := <-errorChan
		if err != nil {
			return nil, err
		}

		if i == 0 {
			ext = val[1]
		}
		if optional == "max" {
			if val[1] > ext {
				ext = val[1]
				index = i
			}
		} else if optional == "min" {
			if val[1] < ext {
				ext = val[1]
				index = i
			}
		} else {
			results[int(val[0])] = val[1]
		}
	}

	if optional != "" {
		var winnerRes PredictWinnerResponse

		winnerRes.Winner = index
		winnerRes.ModelUUID = models[model].UUID
		winnerRes.Prediction = ext

		return winnerRes, nil
	}

	//if optional wasnt set
	predictResponse := &PredictResponse{}
	predictResponse.ModelUUID = models[model].UUID
	predictResponse.Predictions = results

	return predictResponse, nil
}

// helper functions

func respondWithError(w http.ResponseWriter, code int, message string) {
	log.Println("error: ", message)
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
