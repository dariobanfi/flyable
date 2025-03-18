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
	"sync/atomic"

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

const (
	baseURL = "https://de.dhv-xc.de/api/"
)

var (
	api = struct {
		url     string
		token   string
		login   string
		flights string
	}{
		url:     baseURL,
		token:   "xc/login/status",
		login:   "xc/login/login",
		flights: "fli/flights",
	}
	method = struct {
		GET, POST string
	}{
		GET:  "GET",
		POST: "POST",
	}
	client http.Client
	token  string
)

func init() {
	if os.Getenv("XC_DEBUG") == "true" {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		log.SetFlags(log.Ldate | log.Ltime)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Failed to create cookie jar: %v", err)
	}
	client = http.Client{Jar: jar}
}

func jsonMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("JSON marshal error: %v", err)
	}
	return data
}

func jsonUnmarshal[T any](data []byte) T {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("JSON unmarshal error: %v", err)
	}
	return result
}

func httpReq(url string, payload []byte, method string, token string) []byte {
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return body
}

func success(resp map[string]interface{}) bool {
	if _, ok := resp["success"]; ok {
		return true
	}
	return false
}

func getToken(data jsonLogin) string {
	body := httpReq(api.url+api.token, jsonMarshal(data), method.GET, "")
	resp := jsonUnmarshal[map[string]interface{}](body)

	if success, ok := resp["success"].(bool); !ok || !success {
		log.Fatalf("Unable to get token: %v", resp["message"])
	}

	meta := resp["meta"].(map[string]interface{})
	return fmt.Sprintf("%v", meta["token"])
}

func saveIgc(id string, targetdir string) int {
	igcurl := fmt.Sprintf("https://en.dhv-xc.de/flight/%s/igc", id)
	igcdata := httpReq(igcurl, jsonMarshal(""), method.GET, token)

	filename := fmt.Sprintf("%s/%s.igc", targetdir, id)
	if err := os.WriteFile(filename, igcdata, 0644); err != nil {
		log.Printf("ERROR: Failed to save IGC file: %v", err)
		return 0
	}

	log.Printf("INFO: Saved flight: [%s] to: [%s]", id, filename)
	return 1
}

func saveJson(flight Flight, id string, targetdir string) {
	filename := fmt.Sprintf("%s/%s.json", targetdir, id)
	if err := os.WriteFile(filename, jsonMarshal(flight), 0644); err != nil {
		log.Printf("ERROR: Failed to save JSON file: %v", err)
		return
	}
	log.Printf("INFO: Saved flight JSON to: %s", filename)
}

func makedir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalf("Failed to create directory %s: %v", path, err)
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	outputDir := "data"
	makedir(outputDir)

	loginData := jsonLogin{
		User: os.Getenv("XC_USER"),
		Pass: os.Getenv("XC_PASS"),
	}

	token = getToken(loginData)
	log.Printf("INFO: Got token: [%s]", token)

	body := httpReq(api.url+api.login, jsonMarshal(loginData), method.POST, token)
	resp := jsonUnmarshal[map[string]interface{}](body)
	if !success(resp) {
		log.Fatalf("Authentication failed: %v", resp["message"])
	}
	log.Printf("INFO: Logged in")

	var requestUrl = "https://en.dhv-xc.de/api/fli/flights?fkto%5B%5D=9415&fkto%5B%5D=9453&fkto%5B%5D=9438&fkto%5B%5D=13309&fkto%5B%5D=9538&fkto%5B%5D=9136&fkto%5B%5D=9675&fkto%5B%5D=9294&fkto%5B%5D=9410&l-fkto%5B%5D=Brauneck%20(DE)&l-fkto%5B%5D=Hochries%20(DE)&l-fkto%5B%5D=Wank%20(DE)&l-fkto%5B%5D=K%C3%B6ssen%20(AT)&l-fkto%5B%5D=Blomberg%20(DE)&l-fkto%5B%5D=Wallberg%20(DE)&l-fkto%5B%5D=Sulzberg%20(DE)&l-fkto%5B%5D=Hochfelln%20(DE)&l-fkto%5B%5D=Stubaital%20-%20Kreuzjoch%20(AT)&navpars=%7B%22start%22%3A0%2C%22limit%22%3A20%2C%22sort%22%3A%5B%7B%22field%22%3A%22FlightDate%22%2C%22dir%22%3A-1%7D%2C%7B%22field%22%3A%22BestTaskPoints%22%2C%22dir%22%3A-1%7D%5D%7D"

	log.Printf("INFO: %s", requestUrl)

	flights := jsonUnmarshal[FlightResponse](httpReq(requestUrl, jsonMarshal(loginData), method.GET, token))
	var wg sync.WaitGroup
	var saved int32

	for _, flight := range flights.Data {
		saveJson(flight, flight.IDFlight, outputDir)

		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if saveIgc(id, outputDir) == 1 {
				atomic.AddInt32(&saved, 1)
			}
		}(flight.IDFlight)
	}

	wg.Wait()
	log.Printf("INFO: Saved [%d] flights", saved)
}
