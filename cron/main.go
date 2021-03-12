package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
	cron "gopkg.in/robfig/cron.v2"
)

const (
	PullAssessmentGSheet = "pull_assessment_gsheet"
	PullFoo              = "pull"
)

type cronModule struct {
	cronJob *cron.Cron
}

type cronSettingJob struct {
	Func     string `json:"func"`
	Schedule string `json:"schedule"`
	Enable   bool   `json:"enable"`
}

type cronSetting struct {
	Jobs []cronSettingJob `json:"jobs"`
}

func main() {

	jsonFile, err := os.Open("cron-setting.json")
	defer jsonFile.Close()
	if err != nil {
		fmt.Println(err)
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	cronjobConfig := cronSetting{}
	json.Unmarshal(byteValue, &cronjobConfig)

	module := cronModule{}
	module.cronJob = cron.New()

	for _, job := range cronjobConfig.Jobs {
		if job.Enable {
			_, _ = module.cronJob.AddFunc(job.Schedule, func(jobFunc string) func() {
				return func() {
					module.getJob(jobFunc)
				}
			}(job.Func))
		}
	}

	module.cronJob.Start()

	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", helloWorld)

	e.Logger.Fatal(e.Start(":1323"))
}

func helloWorld(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (m *cronModule) getJob(jobFunc string) error {
	fmt.Println("job name: ", jobFunc)
	var cronFunc func()
	switch jobFunc {
	case PullAssessmentGSheet:
		cronFunc = foo
	case PullFoo:
		cronFunc = Pull
	default:
		err := fmt.Errorf("Job %s undefined", jobFunc)
		fmt.Printf("err: %v", err)
		return err
	}

	fmt.Println("running the cron")
	cronFunc()
	return nil
}

func foo() {
	fmt.Println("[CRONJOB] Hello Cron!")
}

func Pull() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	appScope := []string{
		"https://www.googleapis.com/auth/spreadsheets.readonly",
		drive.DriveMetadataReadonlyScope,
	}
	config, err := google.ConfigFromJSON(b, appScope...)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	spreadsheetID := "150UdkVeVGesl3k8-54bNXwQ29vsm8WeQywOWzHHYVw0"
	readRange := "A1:Z"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		titles := []string{}
		for i, row := range resp.Values {
			for colNum, col := range row {
				if i == 0 {
					titles = append(titles, col.(string))
				} else {
					fmt.Printf("%s, %s\n", titles[colNum], col)
				}
			}
		}
	}
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}

	updatedToken, err := config.TokenSource(context.TODO(), tok).Token()

	file, _ := json.MarshalIndent(updatedToken, "", " ")

	_ = ioutil.WriteFile("token.json", file, 0644)

	return config.Client(context.Background(), updatedToken)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
