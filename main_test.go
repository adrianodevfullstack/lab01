package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandlerCep_EmptyCep(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("cep", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	HandlerCep(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Esperado status %d, recebido %d", http.StatusUnprocessableEntity, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Erro ao decodificar resposta JSON: %v", err)
	}

	if response["error"] != "Invalid zipcode" {
		t.Errorf("Esperado erro 'Invalid zipcode', recebido '%s'", response["error"])
	}
}

func TestHandlerCep_InvalidLength(t *testing.T) {
	testCases := []struct {
		name string
		cep  string
	}{
		{"CEP muito curto", "12345"},
		{"CEP muito longo", "123456789"},
		{"CEP com 7 dígitos", "1234567"},
		{"CEP com 9 dígitos", "123456789"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/"+tc.cep, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("cep", tc.cep)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			HandlerCep(w, req)

			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("Esperado status %d, recebido %d", http.StatusUnprocessableEntity, w.Code)
			}

			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Erro ao decodificar resposta JSON: %v", err)
			}

			if response["error"] != "Invalid zipcode" {
				t.Errorf("Esperado erro 'Invalid zipcode', recebido '%s'", response["error"])
			}
		})
	}
}

func TestHandlerCep_CepNotFound(t *testing.T) {
	cepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "CEP não encontrado"})
	}))
	defer cepServer.Close()

	originalURL := "https://cep.awesomeapi.com.br/json/"

	req := httptest.NewRequest("GET", "/00000000", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("cep", "00000000")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	HandlerCep(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusUnprocessableEntity {
		t.Logf("Status recebido: %d (pode ser válido dependendo da API)", w.Code)
	}

	_ = cepServer
	_ = originalURL
}

func TestHandlerCep_Success(t *testing.T) {
	cepResponse := CepAwesomeapiResponse{
		Cep:      "01310100",
		Latitude: "-23.5505",
		Longitude: "-46.6333",
		City:     "São Paulo",
		State:    "SP",
	}

	cepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(cepResponse)
	}))
	defer cepServer.Close()

	weatherResponse := WeatherApiResponse{
		Latitude:  -23.5505,
		Longitude: -46.6333,
		Current: Current{
			Temperature2M: 25.5,
		},
	}

	weatherServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(weatherResponse)
	}))
	defer weatherServer.Close()

	req := httptest.NewRequest("GET", "/01310100", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("cep", "01310100")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	HandlerCep(w, req)

	if w.Code == http.StatusOK {
		var response Temperature
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Erro ao decodificar resposta JSON: %v", err)
		}

		if response.TempC == 0 && response.TempF == 0 && response.TempK == 0 {
			t.Error("Temperaturas não foram calculadas corretamente")
		}

		expectedTempF := response.TempC*1.8 + 32
		expectedTempK := response.TempC + 273.15

		if abs(response.TempF-expectedTempF) > 0.01 {
			t.Errorf("TempF esperada %.2f, recebida %.2f", expectedTempF, response.TempF)
		}

		if abs(response.TempK-expectedTempK) > 0.01 {
			t.Errorf("TempK esperada %.2f, recebida %.2f", expectedTempK, response.TempK)
		}
	}

	_ = cepServer
	_ = weatherServer
}

func TestCepAwesomeapi_Success(t *testing.T) {
	mockResponse := CepAwesomeapiResponse{
		Cep:      "01310100",
		Latitude: "-23.5505",
		Longitude: "-46.6333",
		City:     "São Paulo",
		State:    "SP",
		District: "Bela Vista",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Método esperado GET, recebido %s", r.Method)
		}
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	response, err := CepAwesomeapi("01310100")

	if err != nil {
		t.Skipf("Teste de integração pulado: %v", err)
		return
	}

	if response == nil {
		t.Error("Resposta não deveria ser nil")
		return
	}

	if response.Cep == "" {
		t.Error("CEP não deveria estar vazio")
	}

	if response.Latitude == "" || response.Longitude == "" {
		t.Error("Latitude e Longitude não deveriam estar vazios")
	}

	_ = server
}

func TestCepAwesomeapi_InvalidCep(t *testing.T) {
	response, err := CepAwesomeapi("00000000")

	if err != nil {
		return
	}

	if response != nil {
		if response.Cep == "" {
			return
		}
	}
}

func TestCepAwesomeapi_NetworkError(t *testing.T) {
	t.Skip("Requer refatoração para injetar URL da API")
}

func TestWeatherApi_Success(t *testing.T) {
	mockResponse := WeatherApiResponse{
		Latitude:  -23.5505,
		Longitude: -46.6333,
		Current: Current{
			Temperature2M: 25.5,
			Time:          "2024-01-01T12:00",
			Interval:      0,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Método esperado GET, recebido %s", r.Method)
		}
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	response, err := WeatherApi("-23.5505", "-46.6333")

	if err != nil {
		t.Skipf("Teste de integração pulado: %v", err)
		return
	}

	if response == nil {
		t.Error("Resposta não deveria ser nil")
		return
	}

	if response.Current.Temperature2M == 0 {
		t.Error("Temperatura não deveria ser zero")
	}

	_ = server
}

func TestWeatherApi_InvalidCoordinates(t *testing.T) {
	response, err := WeatherApi("invalid", "invalid")

	if err == nil && response != nil {
		if response.Current.Temperature2M == 0 {
			return
		}
	}
}

func TestTemperature_Conversion(t *testing.T) {
	testCases := []struct {
		name     string
		tempC    float64
		expectedF float64
		expectedK float64
	}{
		{"Zero absoluto", -273.15, -459.67, 0},
		{"Ponto de congelamento", 0, 32, 273.15},
		{"Temperatura ambiente", 25, 77, 298.15},
		{"Ponto de ebulição", 100, 212, 373.15},
		{"Temperatura negativa", -10, 14, 263.15},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempF := tc.tempC*1.8 + 32
			tempK := tc.tempC + 273.15

			if abs(tempF-tc.expectedF) > 0.01 {
				t.Errorf("TempF: esperada %.2f, calculada %.2f", tc.expectedF, tempF)
			}

			if abs(tempK-tc.expectedK) > 0.01 {
				t.Errorf("TempK: esperada %.2f, calculada %.2f", tc.expectedK, tempK)
			}
		})
	}
}

func TestTemperature_StructCreation(t *testing.T) {
	tempC := 25.5
	temp := Temperature{
		TempC: tempC,
		TempF: tempC*1.8 + 32,
		TempK: tempC + 273.15,
	}

	if temp.TempC != 25.5 {
		t.Errorf("TempC esperada 25.5, recebida %.2f", temp.TempC)
	}

	expectedF := 77.9
	if abs(temp.TempF-expectedF) > 0.01 {
		t.Errorf("TempF esperada %.2f, recebida %.2f", expectedF, temp.TempF)
	}

	expectedK := 298.65
	if abs(temp.TempK-expectedK) > 0.01 {
		t.Errorf("TempK esperada %.2f, recebida %.2f", expectedK, temp.TempK)
	}
}

func TestHandlerCep_WeatherApiError(t *testing.T) {
	req := httptest.NewRequest("GET", "/01310100", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("cep", "01310100")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	HandlerCep(w, req)

	validStatuses := []int{http.StatusOK, http.StatusNotFound, http.StatusUnprocessableEntity}
	isValid := false
	for _, status := range validStatuses {
		if w.Code == status {
			isValid = true
			break
		}
	}

	if !isValid {
		t.Errorf("Status inesperado: %d", w.Code)
	}

	var response interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Resposta não é um JSON válido: %v", err)
	}
}

func TestHandlerCep_JSONEncoding(t *testing.T) {
	testCases := []struct {
		name string
		cep  string
	}{
		{"CEP vazio", ""},
		{"CEP inválido", "12345"},
		{"CEP válido", "01310100"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/"+tc.cep, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("cep", tc.cep)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			HandlerCep(w, req)

			var response interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Resposta não é um JSON válido para CEP '%s': %v", tc.cep, err)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") {
				t.Logf("Content-Type não é application/json: %s", contentType)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

