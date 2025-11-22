package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type CepAwesomeapiResponse struct {
	Cep         string `json:"cep"`
	AddressType string `json:"address_type"`
	AddressName string `json:"address_name"`
	Address     string `json:"address"`
	State       string `json:"state"`
	District    string `json:"district"`
	Latitude    string `json:"lat"`
	Longitude   string `json:"lng"`
	City        string `json:"city"`
	Ibge        string `json:"city_ibge"`
	Ddd         string `json:"ddd"`
}

type CurrentUnits struct {
	Time          string `json:"time"`
	Interval      string `json:"interval"`
	Temperature2M string `json:"temperature_2m"`
}

type Current struct {
	Time          string  `json:"time"`
	Interval      int     `json:"interval"`
	Temperature2M float64 `json:"temperature_2m"`
}

type WeatherApiResponse struct {
	Latitude             float64      `json:"latitude"`
	Longitude            float64      `json:"longitude"`
	GenerationtimeMs     float64      `json:"generationtime_ms"`
	UtcOffsetSeconds     int          `json:"utc_offset_seconds"`
	Timezone             string       `json:"timezone"`
	TimezoneAbbreviation string       `json:"timezone_abbreviation"`
	Elevation            float64      `json:"elevation"`
	CurrentUnits         CurrentUnits `json:"current_units"`
	Current              Current      `json:"current"`
}

type Temperature struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func main() {
	r := chi.NewRouter()

	r.Get("/{cep}", HandlerCep)

	fmt.Println("Server is running on port 8080")

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
	}
}

func HandlerCep(w http.ResponseWriter, r *http.Request) {
	cep := chi.URLParam(r, "cep")

	if cep == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid zipcode"})
		return
	}

	if len(cep) != 8 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid zipcode"})
		return
	}

	CepAwesomeapiResponse, error := CepAwesomeapi(cep)
	if error != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Can not find zipcode"})
		return
	}

	WeatherApiResponse, error := WeatherApi(CepAwesomeapiResponse.Latitude, CepAwesomeapiResponse.Longitude)
	if error != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Can not find weather"})
		return
	}

	Temperature := Temperature{
		TempC: WeatherApiResponse.Current.Temperature2M,
		TempF: WeatherApiResponse.Current.Temperature2M*1.8 + 32,
		TempK: WeatherApiResponse.Current.Temperature2M + 273.15,
	}

	json.NewEncoder(w).Encode(Temperature)
}

func CepAwesomeapi(cep string) (*CepAwesomeapiResponse, error) {
	req, err := http.NewRequest("GET", "https://cep.awesomeapi.com.br/json/"+cep, nil)
	if err != nil {
		return nil, err
	}

	resp, error := http.DefaultClient.Do(req)
	if error != nil {
		return nil, error
	}
	defer resp.Body.Close()

	body, error := io.ReadAll(resp.Body)
	if error != nil {
		return nil, error
	}

	var cepAwesomeapiResponse CepAwesomeapiResponse
	error = json.Unmarshal(body, &cepAwesomeapiResponse)
	if error != nil {
		return nil, error
	}
	return &cepAwesomeapiResponse, nil
}

func WeatherApi(latitude string, longitude string) (*WeatherApiResponse, error) {
	req, err := http.NewRequest("GET", "https://api.open-meteo.com/v1/forecast?latitude="+latitude+"&longitude="+longitude+"&current=temperature_2m", nil)
	if err != nil {
		return nil, err
	}

	resp, error := http.DefaultClient.Do(req)
	if error != nil {
		return nil, error
	}
	defer resp.Body.Close()

	body, error := io.ReadAll(resp.Body)
	if error != nil {
		return nil, error
	}

	var weatherApiResponse WeatherApiResponse
	error = json.Unmarshal(body, &weatherApiResponse)
	if error != nil {
		return nil, error
	}
	return &weatherApiResponse, nil
}
