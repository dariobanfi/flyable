package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"sync"

	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

type jsonLogin struct {
	User string `json:"uid"`
	Pass string `json:"pwd"`
}

type Flights struct {
	Data    []FlightInfo `json:"data"`
	Success bool         `json:"success"`
	Message string       `json:"message"`
}

type FlightInfo struct {
	ID                  string `json:"idflight"`
	DATE                string `json:"FlightDate"`
	TakeOff             string `json:"takeofflocation"`
	TakeoffWaypointName string `json:"TakeoffWaypointName"`
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

type Options struct {
	User string `short:"u" long:"user" description:"DHV-XC User name" required:"true"`
	Pass string `short:"p" long:"pass" description:"DHV-XC User Password" required:"true"`
	Dir  string `short:"d" long:"dir" description:"Target directory" required:"true"`
	List bool   `short:"l" long:"list" description:"List flights only, do not download"`
	ID   int    `short:"i" long:"id" description:"Download flight with specific ID only" default:"0"`
}

var client http.Client
var token string

func init() {
	_, ok := os.LookupEnv("XC_DEBUG")
	if ok {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		logrus.Fatalf("Got error while creating cookie jar %s", err.Error())
	}
	client = http.Client{Jar: jar}
}

func json_dumps(data interface{}) []byte {
	payload, err := json.Marshal(data)
	if err != nil {
		logrus.Error("Cant dump json response:")
		logrus.Fatal(err)
	}
	return payload
}

func json_load(data []byte) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		logrus.Error("Cant load json response:")
		logrus.Fatal(err)
	}
	return result
}

func json_loads(data []byte) []interface{} {
	var result []interface{}
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		logrus.Error("Cant load json response:")
		logrus.Fatal(err)
	}
	return result
}

func load_flights(data []byte) Flights {
	var resp Flights
	err := json.Unmarshal([]byte(data), &resp)
	if err != nil {
		logrus.Error("Cant load json response:")
		logrus.Fatal(err)
	}
	return resp
}

func httpReq(url string, payload []byte, method string, token string) []byte {
	logrus.Debug(url)
	logrus.Debugf("Request: [%s]", string(payload))

	var request *http.Request
	if method == Method.POST {
		request, _ = http.NewRequest(method, url, bytes.NewBuffer(payload))
	} else {
		request, _ = http.NewRequest(method, url, nil)
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	if token != "" {
		logrus.Debug("setting token header")
		request.Header.Set("X-CSRF-Token", token)
	}
	response, error := client.Do(request)
	if error != nil {
		logrus.Fatal(error)
	}
	body, _ := ioutil.ReadAll(response.Body)
	logrus.Debugf("Response: [%s]", string(body))

	cookie := fmt.Sprintf("%s", response.Cookies())
	logrus.Debugf("Got cookie: [%s]", cookie)

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
		logrus.Fatalf("Unable to get token: [%s]", resp["message"])
	}
	logrus.Debug(resp)
	meta := resp["meta"].(map[string]interface{})
	logrus.Debug(meta["token"])
	return fmt.Sprintf("%v", meta["token"])
}

func saveIgc(id string, targetdir string) int {
	igcurl := fmt.Sprintf("https://en.dhv-xc.de/flight/%s/igc", id)
	igcdata := httpReq(igcurl, json_dumps(""), Method.GET, token)
	f, _ := os.Create(fmt.Sprintf("%s/%s.igc", targetdir, id))
	logrus.Infof("Saving flight: [%s] to: [%s/%s.igc]", id, targetdir, id)
	f.Write(igcdata)
	f.Close()
	return 1
}

func saveJson(bodyp []byte, id string, targetdir string) {
	err := os.WriteFile(fmt.Sprintf("%s/%s.json", targetdir, id), bodyp, 0644)
	if err != nil {
		logrus.Errorf("Failed to save flights JSON: %v", err)
	}
}

func makedir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0755); os.IsNotExist(err) {
			logrus.Fatalf("Unable to create target dir: [%s]", err)
		}
	}
}

func main() {
	var options Options
	var parser = flags.NewParser(&options, flags.Default)

	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}

	makedir(options.Dir)

	data := jsonLogin{
		User: options.User,
		Pass: options.Pass,
	}

	token = getToken(data)
	logrus.Infof("Got token: [%s]", token)
	body := httpReq(Api.url+Api.login, json_dumps(data), Method.POST, token)
	resp := json_load(body)
	if !success(resp) {
		logrus.Fatalf("Authentication failed: [%s]", resp["message"])
	}
	logrus.Info("Logged in")

	// start := 0
	// paragliding_locations := map[string]string{
	// 	"9415":  "Brauneck (DE)",
	// 	"9453":  "Hochries (DE)",
	// 	"9438":  "Wank (DE)",
	// 	"13309": "Kössen (AT)",
	// 	"9538":  "Blomberg (DE)",
	// 	"9136":  "Wallberg (DE)",
	// 	"9675":  "Sulzberg (DE)",
	// 	"9294":  "Hochfelln (DE)",
	// 	"9410":  "Stubaital - Kreuzjoch (AT)",
	// }
	// navParams := fmt.Sprintf("?navpars={start:%d,limit:20,sort:[{field:FlightDate,dir:-1}]}", start)
	// var filterUrl string
	// for id, _ := range paragliding_locations {
	// 	if filterUrl == "" {
	// 		filterUrl += fmt.Sprintf("fkto[]=%s", id)
	// 	} else {
	// 		filterUrl += fmt.Sprintf("&fkto[]=%s", id)
	// 	}
	// }
	// for _, location := range paragliding_locations {
	// 	filterUrl += fmt.Sprintf("&l-fkto[]=%s", url.QueryEscape(location))
	// }

	var requestUrl = "https://en.dhv-xc.de/api/fli/flights?fkto%5B%5D=9415&fkto%5B%5D=9453&fkto%5B%5D=9438&fkto%5B%5D=13309&fkto%5B%5D=9538&fkto%5B%5D=9136&fkto%5B%5D=9675&fkto%5B%5D=9294&fkto%5B%5D=9410&l-fkto%5B%5D=Brauneck%20(DE)&l-fkto%5B%5D=Hochries%20(DE)&l-fkto%5B%5D=Wank%20(DE)&l-fkto%5B%5D=K%C3%B6ssen%20(AT)&l-fkto%5B%5D=Blomberg%20(DE)&l-fkto%5B%5D=Wallberg%20(DE)&l-fkto%5B%5D=Sulzberg%20(DE)&l-fkto%5B%5D=Hochfelln%20(DE)&l-fkto%5B%5D=Stubaital%20-%20Kreuzjoch%20(AT)&navpars=%7B%22start%22%3A0%2C%22limit%22%3A20%2C%22sort%22%3A%5B%7B%22field%22%3A%22FlightDate%22%2C%22dir%22%3A-1%7D%2C%7B%22field%22%3A%22BestTaskPoints%22%2C%22dir%22%3A-1%7D%5D%7D"

	logrus.Info(requestUrl)

	bodyp := httpReq(requestUrl, json_dumps(data), Method.GET, token)
	flights := load_flights(bodyp)
	var wg sync.WaitGroup
	var saved int = 0
	for i := 0; i < len(flights.Data); i++ {
		if options.List {
			logrus.Infof("Flight ID: [%s] Takeoff: [%s] Date: [%s]",
				flights.Data[i].ID,
				flights.Data[i].TakeoffWaypointName,
				flights.Data[i].DATE,
			)
			continue
		}
		has_id, _ := strconv.Atoi(flights.Data[i].ID)
		saveJson(bodyp, flights.Data[i].ID, options.Dir)

		if options.ID != 0 && has_id != options.ID {
			logrus.Error("Flight ID does not match")
		}
		wg.Add(1)
		go func(id string, date string) {
			defer wg.Done()
			saved += saveIgc(id, options.Dir)
		}(flights.Data[i].ID, flights.Data[i].DATE)
	}
	wg.Wait()
	logrus.Infof("Saved [%d] flights", saved)
}
