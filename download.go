package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/storage"
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
	client        http.Client
	token         string
	storageClient *storage.Client
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

	// Initialize Google Cloud Storage client
	ctx := context.Background()
	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Printf("Warning: Failed to create storage client: %v", err)
	}
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
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
	if err := uploadToCloudStorage(filename); err != nil {
		log.Printf("ERROR: Failed to upload igc to cloud storage: %v", err)
		return 0
	}

	log.Printf("INFO: Saved igc: [%s] to: [%s]", id, filename)
	return 1
}

func saveJson(flight Flight, id string, targetdir string) {
	filename := fmt.Sprintf("%s/%s.json", targetdir, id)
	if err := os.WriteFile(filename, jsonMarshal(flight), 0644); err != nil {
		log.Printf("ERROR: Failed to save JSON file: %v", err)
		return
	}
	if err := uploadToCloudStorage(filename); err != nil {
		log.Printf("ERROR: Failed to upload json to cloud storage: %v", err)
		return
	}
	log.Printf("INFO: Saved flight JSON to: %s", filename)
}

func makedir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalf("Failed to create directory %s: %v", path, err)
	}
}

func uploadToCloudStorage(filename string) error {
	if storageClient == nil {
		return fmt.Errorf("storage client not initialized")
	}

	ctx := context.Background()
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		return fmt.Errorf("BUCKET_NAME environment variable not set")
	}

	bucket := storageClient.Bucket(bucketName)
	objectName := filepath.Base(filename)
	obj := bucket.Object(objectName)
	writer := obj.NewWriter(ctx)

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("failed to copy file to GCS: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer: %v", err)
	}

	log.Printf("INFO: Successfully uploaded %s to GCS", filename)
	return nil
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

	// Define takeoff locations with their IDs
	takeoffLocations := map[string]string{
		"Brauneck (DE)":              "9415",
		"Hochries (DE)":              "9453",
		"Wank (DE)":                  "9438",
		"Kössen (AT)":                "13309",
		"Blomberg (DE)":              "9538",
		"Wallberg (DE)":              "9136",
		"Sulzberg (DE)":              "9675",
		"Hochfelln (DE)":             "9294",
		"Stubaital - Kreuzjoch (AT)": "9410",
	}

	// Build the takeoff location query parameters
	var fktoParams []string
	var locationParams []string
	for location, id := range takeoffLocations {
		fktoParams = append(fktoParams, "fkto[]="+id)
		locationParams = append(locationParams, "l-fkto[]="+url.QueryEscape(location))
	}

	// Define navigation parameters
	navParameters := map[string]interface{}{
		"start": 0,
		"limit": 500,
		"sort": []map[string]interface{}{
			{"field": "FlightDate", "dir": -1},
			{"field": "BestTaskPoints", "dir": -1},
		},
	}

	//  As of 18.03.2025, there are 91122 flights in the database, no need to worry for the older ones
	totalFlights := 91122

	for start := 0; start < totalFlights; start += 500 {
		navParameters["start"] = start

		// Construct the full URL for this batch
		baseUrl := "https://en.dhv-xc.de/api/fli/flights"
		navParamsJson, _ := json.Marshal(navParameters)
		queryParams := append(fktoParams, locationParams...)
		queryParams = append(queryParams, "navpars="+url.QueryEscape(string(navParamsJson)))

		requestUrl := baseUrl + "?" + strings.Join(queryParams, "&")

		log.Printf("INFO: Fetching flights %d to %d", start, start+500)
		log.Printf("INFO: %s", requestUrl)

		batchFlights := jsonUnmarshal[FlightResponse](httpReq(requestUrl, jsonMarshal(loginData), method.GET, token))

		var wg sync.WaitGroup
		var saved int32

		for _, flight := range batchFlights.Data {
			wg.Add(1)
			go func(flight Flight, id string) {
				defer wg.Done()
				saveJson(flight, id, outputDir)
			}(flight, flight.IDFlight)

			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				if saveIgc(id, outputDir) == 1 {
					atomic.AddInt32(&saved, 1)
				}
			}(flight.IDFlight)
		}

		wg.Wait()

		log.Printf("INFO: Sleeping for 200 ms, currently at %d", start)
		time.Sleep(5000 * time.Millisecond)

	}

}
