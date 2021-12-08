package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type ModelResponse struct {
	Model  string  `json:"model"`
	UUID   string  `json:"uuid"`
	SizeMB float64 `json:"sizeMB"`
}

type SamplesRequest struct {
	Features [][]interface{} `json:"features"`
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

func httpLoadModels(w http.ResponseWriter, r *http.Request) {

	modelsRes, code, err := apiLoadModels()
	if err != nil {
		respondWithError(w, code, err.Error())
	}

	err = respondWithJSON(w, code, modelsRes)
	if err != nil {
		log.Println(err)
	}
}

func httpMakePrediction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if vars["model"] == "" {
		respondWithError(w, http.StatusBadRequest, "Missing model param.")
		return
	}

	if models[vars["model"]] == nil {
		respondWithError(w, http.StatusInternalServerError, "Model not loaded in memory.")
		return
	}
	defer r.Body.Close()

	//decode request
	var req SamplesRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil || len(req.Features) == 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		log.Println(err)
		return
	}

	response, err := apiMakePrediction(&req, vars["model"], vars["optional"])
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error occurred during prediction.")
		log.Println(err)
		return
	}
	respondWithJSON(w, http.StatusOK, response)
}
