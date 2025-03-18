package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Meta struct {
	TotalCount int `json:"totalCount"`
}

type FlightResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Meta    Meta     `json:"meta"`
	Data    []Flight `json:"data"`
}

type Flight struct {
	IDFlight                  string  `json:"IDFlight"`
	FKGliderCategory          string  `json:"FKGliderCategory"`
	Category                  string  `json:"Category"`
	FKCompetitionClass        string  `json:"FKCompetitionClass"`
	FKCompetitionClassDesired *string `json:"FKCompetitionClassDesired"`
	CompetitionClass          string  `json:"CompetitionClass"`
	FKLaunchtype              string  `json:"FKLaunchtype"`
	Launchtype                string  `json:"Launchtype"`
	FKPilot                   string  `json:"FKPilot"`
	FirstName                 string  `json:"FirstName"`
	LastName                  string  `json:"LastName"`
	Nationality               string  `json:"Nationality"`
	FKFederation              *string `json:"FKFederation"`
	ClubID                    *string `json:"ClubID"`
	ClubName                  *string `json:"ClubName"`
	Glider                    string  `json:"Glider"`
	FKGlider                  string  `json:"FKGlider"`
	FKGliderBrand             string  `json:"FKGliderBrand"`
	GliderBrand               string  `json:"GliderBrand"`
	GliderLogo                *string `json:"GliderLogo"`
	FKGliderClassification    string  `json:"FKGliderClassification"`
	GliderClassification      string  `json:"GliderClassification"`
	FKSeason                  string  `json:"FKSeason"`
	FlightDate                string  `json:"FlightDate"`
	UtcOffset                 string  `json:"UtcOffset"`
	FlightStartTime           string  `json:"FlightStartTime"`
	FlightEndTime             string  `json:"FlightEndTime"`
	FlightDuration            string  `json:"FlightDuration"`
	FirstLat                  string  `json:"FirstLat"`
	FirstLng                  string  `json:"FirstLng"`
	LastLat                   string  `json:"LastLat"`
	LastLng                   string  `json:"LastLng"`
	FlightMinLat              string  `json:"FlightMinLat"`
	FlightMaxLat              string  `json:"FlightMaxLat"`
	FlightMinLng              string  `json:"FlightMinLng"`
	FlightMaxLng              string  `json:"FlightMaxLng"`
	TakeoffCountry            string  `json:"TakeoffCountry"`
	FKTakeoffWaypoint         string  `json:"FKTakeoffWaypoint"`
	TakeoffWaypointOffset     string  `json:"TakeoffWaypointOffset"`
	TakeoffLocation           string  `json:"TakeoffLocation"`
	TakeoffWaypointName       string  `json:"TakeoffWaypointName"`
	FKClosestWaypoint         string  `json:"FKClosestWaypoint"`
	ClosestWaypointOffset     string  `json:"ClosestWaypointOffset"`
	LandingCountry            string  `json:"LandingCountry"`
	FKLandingWaypoint         string  `json:"FKLandingWaypoint"`
	LandingWaypointOffset     string  `json:"LandingWaypointOffset"`
	LandingWaypointName       string  `json:"LandingWaypointName"`
	LandingLocation           string  `json:"LandingLocation"`
	LinearDistance            string  `json:"LinearDistance"`
	MaxLinearDistance         string  `json:"MaxLinearDistance"`
	ArcDistance               string  `json:"ArcDistance"`
	FKBestTaskType            string  `json:"FKBestTaskType"`
	BestTaskType              string  `json:"BestTaskType"`
	BestTaskTypeKey           string  `json:"BestTaskTypeKey"`
	BestTaskDistance          string  `json:"BestTaskDistance"`
	BestTaskPoints            string  `json:"BestTaskPoints"`
	BestTaskDuration          string  `json:"BestTaskDuration"`
	MaxSpeed                  string  `json:"MaxSpeed"`
	GroundSpeed               string  `json:"GroundSpeed"`
	BestTaskSpeed             string  `json:"BestTaskSpeed"`
	TakeoffAltitude           string  `json:"TakeoffAltitude"`
	MaxAltitude               string  `json:"MaxAltitude"`
	MinAltitude               string  `json:"MinAltitude"`
	ElevationGain             string  `json:"ElevationGain"`
	MeanAltitudeDiff          string  `json:"MeanAltitudeDiff"`
	MaxClimb                  string  `json:"MaxClimb"`
	MinClimb                  string  `json:"MinClimb"`
	CommentsEnabled           string  `json:"CommentsEnabled"`
	CountComments             string  `json:"CountComments"`
	HasPhotos                 string  `json:"HasPhotos"`
	IsBigSmileCandidate       string  `json:"IsBigSmileCandidate"`
	GRecordStatus             string  `json:"GRecordStatus"`
	StatisticsValid           string  `json:"StatisticsValid"`
	IsNew                     string  `json:"IsNew"`
	IgcFilename               string  `json:"IgcFilename"`
	Dataversion               string  `json:"Dataversion"`
}

type jsonLogin struct {
	User string `json:"uid"`
	Pass string `json:"pwd"`
}

var Api = struct {
	url     string
	token   string
	login   string
	flights string
}{
	url:     "https://de.dhv-xc.de/api/",
	token:   "xc/login/status",
	login:   "xc/login/login",
	flights: "fli/flights",
}
var Method = struct {
	GET  string
	POST string
}{
	GET:  "GET",
	POST: "POST",
}

var client http.Client
var token string

func init() {
	_, ok := os.LookupEnv("XC_DEBUG")
	if ok {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		log.SetFlags(log.Ldate | log.Ltime)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Got error while creating cookie jar %s", err.Error())
	}
	client = http.Client{Jar: jar}
}

func json_dumps(data interface{}) []byte {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("ERROR: Cant dump json response: %v", err)
		log.Fatal(err)
	}
	return payload
}

func json_load(data []byte) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		log.Printf("ERROR: Cant load json response: %v", err)
		log.Fatal(err)
	}
	return result
}

func unmarshalFlightResponse(data []byte) FlightResponse {
	var resp FlightResponse
	err := json.Unmarshal([]byte(data), &resp)
	if err != nil {
		log.Printf("ERROR: Cant load json response: %v", err)
		log.Fatal(err)
	}
	return resp
}

func httpReq(url string, payload []byte, method string, token string) []byte {
	var request *http.Request
	if method == Method.POST {
		request, _ = http.NewRequest(method, url, bytes.NewBuffer(payload))
	} else {
		request, _ = http.NewRequest(method, url, nil)
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	if token != "" {
		request.Header.Set("X-CSRF-Token", token)
	}
	response, error := client.Do(request)
	if error != nil {
		log.Fatal(error)
	}
	body, _ := io.ReadAll(response.Body)

	return body
}

func success(resp map[string]interface{}) bool {
	if _, ok := resp["success"]; ok {
		return true
	}
	return false
}

func getToken(data jsonLogin) string {
	body := httpReq(Api.url+Api.token, json_dumps(data), Method.GET, "")
	resp := json_load(body)
	if !success(resp) {
		log.Fatalf("Unable to get token: [%s]", resp["message"])
	}
	log.Printf("DEBUG: %v", resp)
	meta := resp["meta"].(map[string]interface{})
	log.Printf("DEBUG: %v", meta["token"])
	return fmt.Sprintf("%v", meta["token"])
}

func saveIgc(id string, targetdir string) int {
	igcurl := fmt.Sprintf("https://en.dhv-xc.de/flight/%s/igc", id)
	igcdata := httpReq(igcurl, json_dumps(""), Method.GET, token)
	f, _ := os.Create(fmt.Sprintf("%s/%s.igc", targetdir, id))
	log.Printf("INFO: Saving flight: [%s] to: [%s/%s.igc]", id, targetdir, id)
	f.Write(igcdata)
	f.Close()
	return 1
}

func saveJson(flight Flight, id string, targetdir string) {
	jsonData, err := json.MarshalIndent(flight, "", "    ")
	if err != nil {
		log.Printf("ERROR: Failed to marshal flight data: %v", err)
		return
	}

	filename := fmt.Sprintf("%s/%s.json", targetdir, id)
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		log.Printf("ERROR: Failed to write JSON file: %v", err)
		return
	}

	log.Printf("INFO: Saved flight JSON to: %s", filename)
}

func makedir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0755); os.IsNotExist(err) {
			log.Fatalf("Unable to create target dir: [%s]", err)
		}
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var outputDir = "data"
	makedir(outputDir)

	data := jsonLogin{
		User: os.Getenv("XC_USER"),
		Pass: os.Getenv("XC_PASS"),
	}

	token = getToken(data)
	log.Printf("INFO: Got token: [%s]", token)
	body := httpReq(Api.url+Api.login, json_dumps(data), Method.POST, token)
	resp := json_load(body)
	if !success(resp) {
		log.Fatalf("Authentication failed: [%s]", resp["message"])
	}
	log.Printf("INFO: Logged in")

	var requestUrl = "https://en.dhv-xc.de/api/fli/flights?fkto%5B%5D=9415&fkto%5B%5D=9453&fkto%5B%5D=9438&fkto%5B%5D=13309&fkto%5B%5D=9538&fkto%5B%5D=9136&fkto%5B%5D=9675&fkto%5B%5D=9294&fkto%5B%5D=9410&l-fkto%5B%5D=Brauneck%20(DE)&l-fkto%5B%5D=Hochries%20(DE)&l-fkto%5B%5D=Wank%20(DE)&l-fkto%5B%5D=K%C3%B6ssen%20(AT)&l-fkto%5B%5D=Blomberg%20(DE)&l-fkto%5B%5D=Wallberg%20(DE)&l-fkto%5B%5D=Sulzberg%20(DE)&l-fkto%5B%5D=Hochfelln%20(DE)&l-fkto%5B%5D=Stubaital%20-%20Kreuzjoch%20(AT)&navpars=%7B%22start%22%3A0%2C%22limit%22%3A20%2C%22sort%22%3A%5B%7B%22field%22%3A%22FlightDate%22%2C%22dir%22%3A-1%7D%2C%7B%22field%22%3A%22BestTaskPoints%22%2C%22dir%22%3A-1%7D%5D%7D"

	log.Printf("INFO: %s", requestUrl)

	bodyp := httpReq(requestUrl, json_dumps(data), Method.GET, token)
	flights := unmarshalFlightResponse(bodyp)
	var wg sync.WaitGroup
	var saved int = 0
	for i := 0; i < len(flights.Data); i++ {
		saveJson(flights.Data[i], flights.Data[i].IDFlight, outputDir)

		wg.Add(1)
		go func(id string, date string) {
			defer wg.Done()
			saved += saveIgc(id, outputDir)
		}(flights.Data[i].IDFlight, flights.Data[i].FlightDate)
	}
	wg.Wait()
	log.Printf("INFO: Saved [%d] flights", saved)
}
