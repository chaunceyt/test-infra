package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/google/go-github/github"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var rootCmd = &cobra.Command{
	Use:   "flake-tracker",
	Short: "Creates a point-in-time report listing on Jobs reported as FLAKY in the TestGridSummary for sig-release-blocking and sig-release-informing",
	Run: func(cmd *cobra.Command, args []string) {
		tgInforming := &TabGroupStatus{
			Name:               "sig-release-master-informing",
			CollectedAt:        time.Now(),
			TabGroupSummaryUrl: fmt.Sprintf(TG_TABGROUP_SUMMARY_FMT, "sig-release-master-informing"),
		}
		tgBlocking := &TabGroupStatus{
			Name:               "sig-release-master-blocking",
			CollectedAt:        time.Now(),
			TabGroupSummaryUrl: fmt.Sprintf(TG_TABGROUP_SUMMARY_FMT, "sig-release-master-blocking"),
		}
		output, _ := cmd.Flags().GetString("output")

		if output == "json" {
			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			s.Start()
			generateJSONOutput(tgBlocking)
			generateJSONOutput(tgInforming)
			s.Stop()

		} else if output == "dashboard" {
			dashboard(tgBlocking, tgInforming)

		} else {
			runReport(tgBlocking)
			runReport(tgInforming)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringP("output", "o", "", "the format of output csv, json")
}

const (
	TG_TABGROUP_SUMMARY_FMT string = "https://testgrid.k8s.io/%s/summary"
	TG_JOB_TEST_TABLE_FMT   string = "https://testgrid.k8s.io/%s/table?tab=%s&width=5&exclude-non-failed-tests=&sort-by-flakiness=&dashboard=%s"
	ciSignalBoardId                = 2093513
	masterBlockingJSONFile         = "json-sig-release-master-blocking.json"
	masterInformingJSONFile        = "json-sig-release-master-informing.json"
)

var reportFields log.Fields

type reportPage struct {
	PageTitle   string
	ReportItems []reportItem
}

type reportItem struct {
	Collected string `json:"collected"`
	JobName   string `json:"jobName"`
	OwnerName string `json:"ownerName"`
	Status    string `json:"status"`
	TestName  string `json:"testName"`
	TestURL   string `json:"testURL"`
	Rtype     string `json:"type"`
}

// Tracks status of Jobs for a single TestGrid TabGroup
type TabGroupStatus struct {
	Name               string
	CollectedAt        time.Time
	Count              int
	TabGroupSummaryUrl string
	JobIssues          map[string]issue
	FlakingJobs        map[string]jobStatus
	PassingJobs        map[string]jobStatus
	FailedJobs         map[string]jobStatus
}

type issue struct {
	org         string
	issue       string
	repo        string
	board       string
	status      string
	lastupdate  string
	created     time.Time
	lastupdated time.Time
	fix         pr
	evidence    []string // Evidenciary URLs from Prow, Testgrid and Triage for flakes
}

type pr struct {
	id, status string
}

// Status of job. jobName is the key in the TabGroupStatus Jobs maps
type jobStatus struct {
	OverallStatus           string             `json:"overall_status"`
	Alert                   string             `json:"alert"`
	LastRun                 int64              `json:"last_run_timestamp"`
	LastUpdate              int64              `json:"last_update_timestamp"`
	LatestGreenRun          string             `json:"latest_green"`
	LatestStatusIcon        string             `json:"overall_status_icon"`
	LatestStatusDescription string             `json:"status"`
	url                     string             // Url for testGridJobResult
	jobTestResults          *testGridJobResult // See CollectFlakyTests
}

type testGridJobResult struct {
	TestGroupName string `json:"test-group-name"`
	/* Unused fields. Reviewers can ignore for now.
	           Left in as comment for possible future report extention
		Query         string `json:"query"`
		Status        string `json:"status"`
		PhaseTimer    struct {
			Phases []string  `json:"phases"`
			Delta  []float64 `json:"delta"`
			Total  float64   `json:"total"`
		} `json:"phase-timer"`
		Cached  bool   `json:"cached"`
		Summary string `json:"summary"`
		Bugs    struct {
		} `json:"bugs"`
		Changelists       []string   `json:"changelists"`
		ColumnIds         []string   `json:"column_ids"`
		CustomColumns     [][]string `json:"custom-columns"`
		ColumnHeaderNames []string   `json:"column-header-names"`
		Groups            []string   `json:"groups"`
		Metrics           []string   `json:"metrics"`
	*/
	Tests []struct {
		Name         string        `json:"name"`
		OriginalName string        `json:"original-name"`
		Alert        interface{}   `json:"alert"`
		LinkedBugs   []interface{} `json:"linked_bugs"`
		Messages     []string      `json:"messages"`
		ShortTexts   []string      `json:"short_texts"`
		Statuses     []struct {
			Count int `json:"count"`
			Value int `json:"value"`
		} `json:"statuses"`
		Target       string      `json:"target"`
		UserProperty interface{} `json:"user_property"`
		// Calculated Field added here
		sig string
	} `json:"tests"`
	/* Unused fields
		RowIds       []string    `json:"row_ids"`
		Timestamps   []int64     `json:"timestamps"`
		Clusters     interface{} `json:"clusters"`
		TestIDMap    interface{} `json:"test_id_map"`
		TestMetadata struct {
		} `json:"test-metadata"`
		StaleTestThreshold    int    `json:"stale-test-threshold"`
		NumStaleTests         int    `json:"num-stale-tests"`
		AddTabularNamesOption bool   `json:"add-tabular-names-option"`
		ShowTabularNames      bool   `json:"show-tabular-names"`
		Description           string `json:"description"`
		BugComponent          int    `json:"bug-component"`
		CodeSearchPath        string `json:"code-search-path"`
		OpenTestTemplate      struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"open-test-template"`
		FileBugTemplate struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
				Body  string `json:"body"`
				Title string `json:"title"`
			} `json:"options"`
		} `json:"file-bug-template"`
		AttachBugTemplate struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"attach-bug-template"`
		ResultsURLTemplate struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"results-url-template"`
		CodeSearchURLTemplate struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"code-search-url-template"`
		AboutDashboardURL string `json:"about-dashboard-url"`
		OpenBugTemplate   struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"open-bug-template"`
		ContextMenuTemplate struct { // jobStatus{
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"context-menu-template"`
	ResultsText   string      `json:"results-text"`
	LatestGreen   string      `json:"latest-green"`
	TriageEnabled bool        `json:"triage-enabled"`
	Notifications interface{} `json:"notifications"`
	OverallStatus int         `json:"overall-status"`
	*/
}

// CollectFlakyTests adds a list of Tests that are flaking for each Flaky Job
func (t *TabGroupStatus) CollectFlakyTests() error {

	for jobName := range t.FlakingJobs {
		url := fmt.Sprintf(TG_JOB_TEST_TABLE_FMT,
			t.Name, url.QueryEscape(jobName), t.Name)
		resp, err := http.Get(url)
		if err != nil {
			t.logError("HTTP get job test results", err, url)
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		var flakingTestResults testGridJobResult
		err = json.Unmarshal(body, &flakingTestResults)
		if err != nil {
			t.logError("Unmarshalling Test Result", err, url)
			return err
		}
		// SearchLoggedIssues()
		// Store data and url where we found it. tmp var used as per
		// https://github.com/golang/go/issues/3117#issuecomment-66063615
		var tmp = t.FlakingJobs[jobName]
		addSigToTestResults(&flakingTestResults)
		tmp.jobTestResults = &flakingTestResults
		tmp.url = url
		t.FlakingJobs[jobName] = tmp
	}
	return nil
}

// CollectIssuesFromBoard retrieves logged issues from the user-supplied board
// populating maps that associate them with the CI Jobs (and tests) where flakes occured
func (t *TabGroupStatus) CollectIssuesFromBoard() {

	githubApiToken := os.Getenv("GITHUB_AUTH_TOKEN")
	if githubApiToken == "" {
		log.Error("GITHUB_API_TOKEN is not set in the process env. Use export GIHUB_API_TOKEN")
		panic("Quitting")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubApiToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	rl, _, e := client.RateLimits(ctx)

	if _, ok := e.(*github.RateLimitError); ok {
		log.Error(rl)
		panic("Github client Rate Limit reached")
	}

	opt := &github.ProjectCardListOptions{}
	listOpt := &github.ListOptions{}
	cols, r, err := client.Projects.ListProjectColumns(ctx, ciSignalBoardId, listOpt)
	log.Infof("c.P.LPC cols %v\n", cols)

	if err != nil {
		log.Error(err)
		log.Error(r)
		panic("Github client could not get")
	}
	for _, col := range cols {
		cards, _, err := client.Projects.ListProjectCards(ctx, *col.ID, opt)

		if err != nil {
			log.Error(err)
			log.Error(r)
			panic("Github client could not get")
		}

		for i, card := range cards {
			fmt.Printf("col[%d] card URL : %+v \n", i, card)
			fmt.Printf("col[%d] GetContentUrl : %s\n", i, card.GetContentURL())
			fmt.Printf("col[%d] GetUrl: %s\n", i, card.GetURL())
			fmt.Printf("col[%d] CreatedBy : %s\n", i, card.GetCreator().GetLogin())
			flake, err := getIssueDetail(client, card.GetContentURL())
			if err != nil {
				log.Errorf("flake-tracker getIssueDetail()\n%v\n", err)
			}
			fmt.Printf("Flake title: %v\n", flake)
		}
	}
}
func getIssueDetail(client *github.Client, jobSummaryUrl string) (*github.Issue, error) {

	urlParts := strings.Split(jobSummaryUrl, "/")
	log.Info(urlParts)
	i := urlParts[len(urlParts)-1]
	r := urlParts[len(urlParts)-3]
	o := urlParts[len(urlParts)-4]

	issueNumber, err := strconv.Atoi(i)
	if err != nil {
		return nil, err
	}
	ghIssue, _, err := client.Issues.Get(context.Background(), o, r, issueNumber)

	if err != nil {
		return nil, err
	}
	return ghIssue, nil
}

// CollectFailedTests adds a list of Tests that are failing for each Failed Job
func (t *TabGroupStatus) CollectFailedTests() error {

	for jobName := range t.FailedJobs {
		url := fmt.Sprintf(TG_JOB_TEST_TABLE_FMT,
			t.Name, url.QueryEscape(jobName), t.Name)
		resp, err := http.Get(url)
		if err != nil {
			t.logError("HTTP get Failed job test results", err, url)
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		var failedTestResults testGridJobResult
		err = json.Unmarshal(body, &failedTestResults)
		if err != nil {
			t.logError("Unmarshalling Failed Test Result", err, url)
			return err
		}
		// Store data and url where we found it. tmp var used as per
		// https://github.com/golang/go/issues/3117#issuecomment-66063615
		var tmp = t.FailedJobs[jobName]
		addSigToTestResults(&failedTestResults)
		tmp.jobTestResults = &failedTestResults
		tmp.url = url
		t.FailedJobs[jobName] = tmp
	}
	return nil
}

// addSigToTestResults sets the sig field on tgJobResult using the test name
// by finding the first occurance of [sig-SIGNAME], if no sig is found sets sig
// to "job-owner"
func addSigToTestResults(tgJobResult *testGridJobResult) {
	var sigRe = regexp.MustCompile(`\[sig-.+?\] `)
	for i, t := range tgJobResult.Tests {
		sig := sigRe.FindString(t.Name)
		if sig != "" {
			tgJobResult.Tests[i].sig = sig
		} else {
			tgJobResult.Tests[i].sig = "job-owner"
		}
	}
	return
}

// CollectStatus populates t with job status summary data from TestGrid
func (t *TabGroupStatus) CollectStatus() error {
	resp, err := http.Get(t.TabGroupSummaryUrl)
	if err != nil {
		t.logError("HTTP getting", err)
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.logError("Reading HTTP response buffer", err)
		return err
	}

	jobs := make(map[string]jobStatus)

	err = json.Unmarshal(body, &jobs)
	if err != nil {
		t.logError("UnMarshalling reponse body", err)
		return err
	}

	t.FlakingJobs = make(map[string]jobStatus, 0)

	for name, job := range jobs {
		if job.OverallStatus == "FLAKY" {
			t.FlakingJobs[name] = jobs[name]
		}
	}

	t.FailedJobs = make(map[string]jobStatus, 0)
	for name, job := range jobs {
		if job.OverallStatus == "FAILING" {
			t.FailedJobs[name] = job
		}
	}

	t.PassingJobs = make(map[string]jobStatus, 0)
	for name, job := range jobs {
		if job.OverallStatus == "PASSING" {
			t.PassingJobs[name] = job
		}
	}

	t.Count = len(jobs)
	return nil
}

func (t *TabGroupStatus) logError(action string, err error, fields ...string) log.Logger {
	var augmentedLogger = log.WithFields(reportFields).WithField(
		"ACTION", action,
	)
	for i, field := range fields {
		augmentedLogger.WithField(fmt.Sprintf("ExtraField %d", i), field)
	}
	augmentedLogger.Error("%v", err)

	return log.Logger{}
}

func runReport(tabGroupStatus *TabGroupStatus) {

	log.SetFormatter(&log.JSONFormatter{})
	reportFields = log.Fields{
		"DATA BEING RETRIEVED": "Job Status Summary TestGrid TabGroup",
		"TEST_GRID TAB_GROUP":  tabGroupStatus.Name,
		"COLLECTION TIME":      tabGroupStatus.CollectedAt,
		"TB GRP SMMRY URL":     tabGroupStatus.TabGroupSummaryUrl,
	}

	tabGroupStatus.CollectStatus()
	tabGroupStatus.CollectFlakyTests()
	tabGroupStatus.CollectFailedTests()

	for jobName, jobStatus := range tabGroupStatus.FlakingJobs {
		jobFlakyTests := jobStatus.jobTestResults
		for _, flakyTest := range jobFlakyTests.Tests {

			fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n",
				tabGroupStatus.CollectedAt.Format(time.UnixDate),
				jobStatus.OverallStatus, jobName, flakyTest.sig,
				flakyTest.Name, jobStatus.url)

		}
	}

	for jobName, jobStatus := range tabGroupStatus.FailedJobs {
		jobFailedTests := jobStatus.jobTestResults
		for _, failedTest := range jobFailedTests.Tests {

			fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n",
				tabGroupStatus.CollectedAt.Format(time.UnixDate),
				jobStatus.OverallStatus, jobName, failedTest.sig,
				failedTest.Name, jobStatus.url)
		}
	}

	for jobName, jobStatus := range tabGroupStatus.PassingJobs {

		fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n",
			tabGroupStatus.CollectedAt.Format(time.UnixDate),
			jobStatus.OverallStatus, jobName, "", "", jobStatus.url)
	}
}

// generateJSONOutput - generate json files
func generateJSONOutput(tabGroupStatus *TabGroupStatus) {

	log.SetFormatter(&log.JSONFormatter{})
	reportFields = log.Fields{
		"DATA BEING RETRIEVED": "Job Status Summary TestGrid TabGroup",
		"TEST_GRID TAB_GROUP":  tabGroupStatus.Name,
		"COLLECTION TIME":      tabGroupStatus.CollectedAt,
		"TB GRP SMMRY URL":     tabGroupStatus.TabGroupSummaryUrl,
	}

	tabGroupStatus.CollectStatus()
	tabGroupStatus.CollectFlakyTests()
	tabGroupStatus.CollectFailedTests()

	jsonOutput := make([]map[string]interface{}, 0, 0)

	flakyTestData := make([]map[string]interface{}, 0, 0)
	for jobName, jobStatus := range tabGroupStatus.FlakingJobs {
		jobFlakyTests := jobStatus.jobTestResults
		for _, flakyTest := range jobFlakyTests.Tests {
			flakyTestReport := make(map[string]interface{})
			flakyTestReport["type"] = "flakyTest"
			flakyTestReport["collected"] = tabGroupStatus.CollectedAt.Format(time.UnixDate)
			flakyTestReport["status"] = jobStatus.OverallStatus
			flakyTestReport["jobName"] = jobName
			flakyTestReport["ownerName"] = flakyTest.sig
			flakyTestReport["testName"] = flakyTest.Name
			flakyTestReport["testURL"] = jobStatus.url
			flakyTestData = append(flakyTestData, flakyTestReport)
		}
	}
	jsonOutput = append(jsonOutput, flakyTestData...)

	failedTestData := make([]map[string]interface{}, 0, 0)
	for jobName, jobStatus := range tabGroupStatus.FailedJobs {
		jobFailedTests := jobStatus.jobTestResults
		for _, failedTest := range jobFailedTests.Tests {
			failedTestReport := make(map[string]interface{})
			failedTestReport["type"] = "failedTest"
			failedTestReport["collected"] = tabGroupStatus.CollectedAt.Format(time.UnixDate)
			failedTestReport["status"] = jobStatus.OverallStatus
			failedTestReport["jobName"] = jobName
			failedTestReport["ownerName"] = failedTest.sig
			failedTestReport["testName"] = failedTest.Name
			failedTestReport["testURL"] = jobStatus.url
			failedTestData = append(failedTestData, failedTestReport)
		}
	}
	jsonOutput = append(jsonOutput, failedTestData...)

	jobStatusData := make([]map[string]interface{}, 0, 0)
	for jobName, jobStatus := range tabGroupStatus.PassingJobs {
		jobStatusReport := make(map[string]interface{})
		jobStatusReport["type"] = "jobStatus"
		jobStatusReport["collected"] = tabGroupStatus.CollectedAt.Format(time.UnixDate)
		jobStatusReport["status"] = jobStatus.OverallStatus
		jobStatusReport["jobName"] = jobName
		jobStatusReport["ownerName"] = ""
		jobStatusReport["testName"] = ""
		jobStatusReport["testURL"] = jobStatus.url
		jobStatusData = append(jobStatusData, jobStatusReport)
	}
	jsonOutput = append(jsonOutput, jobStatusData...)

	jsonOutputBytes, err := json.MarshalIndent(jsonOutput, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	r := bytes.NewReader(jsonOutputBytes)
	if b, err := ioutil.ReadAll(r); err == nil {
		jsonFilename := "json-" + tabGroupStatus.Name + ".json"
		_ = ioutil.WriteFile(jsonFilename, b, 0644)
	}
}

// loadJSONData - load json file from filesystem.
func loadJSONData(jsonFile string) string {

	jsReportData, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		panic(err)
	}
	return string(jsReportData)

}

// dashboard - display report data in default browser on random port.
func dashboard(tgBlocking *TabGroupStatus, tgInforming *TabGroupStatus) {
	log.Println("Generating report files...")
	generateJSONOutput(tgBlocking)
	generateJSONOutput(tgInforming)

	log.Println("Starting dashboard application...")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("index").Parse(indexTemplate)
		if err != nil {
			log.Fatal(err)
		}

		data := reportPage{
			PageTitle: "Master Blocking and Informing report",
		}
		tmpl.Execute(w, data)
	})

	http.HandleFunc("/generate-report-files", func(w http.ResponseWriter, r *http.Request) {
		generateJSONOutput(tgBlocking)
		generateJSONOutput(tgInforming)
		http.Redirect(w, r, "/", 302)

	})

	http.HandleFunc("/master-blocking", func(w http.ResponseWriter, r *http.Request) {
		masterBlockingData := loadJSONData(masterBlockingJSONFile)
		bsMasterBlockingData := []byte(masterBlockingData)

		var reportData []reportItem
		err := json.Unmarshal(bsMasterBlockingData, &reportData)
		if err != nil {
			log.Fatal(err)
		}

		tmpl, err := template.New("report").Parse(reportTemplate)
		if err != nil {
			log.Fatal(err)
		}

		data := reportPage{
			PageTitle:   "Master Blocking Report",
			ReportItems: reportData,
		}

		tmpl.Execute(w, data)
	})

	http.HandleFunc("/master-informing", func(w http.ResponseWriter, r *http.Request) {
		masterInformingData := loadJSONData(masterInformingJSONFile)
		bsMasterInformingData := []byte(masterInformingData)

		var reportData []reportItem
		err := json.Unmarshal(bsMasterInformingData, &reportData)
		if err != nil {
			log.Fatal(err)
		}

		tmpl, err := template.New("report").Parse(reportTemplate)
		if err != nil {
			log.Fatal(err)
		}

		data := reportPage{
			PageTitle:   "Master Informing Report",
			ReportItems: reportData,
		}

		tmpl.Execute(w, data)
	})

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	var port = fmt.Sprintf(":%d", listener.Addr().(*net.TCPAddr).Port)

	appURL := fmt.Sprintf("%s%s", "http://localhost", port)
	s := http.Server{Addr: port}
	go func() {
		log.Fatal(s.Serve(listener))
	}()

	browser.OpenURL(appURL)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutdown signal received, exiting...")

	s.Shutdown(context.Background())

}
