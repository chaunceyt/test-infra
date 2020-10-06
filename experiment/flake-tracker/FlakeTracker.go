package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const TG_TABGROUP_SUMMARY_FMT string = "https://testgrid.k8s.io/%s/summary"
const TG_JOB_TEST_TABLE_FMT string = "https://testgrid.k8s.io/%s/table?tab=%s&width=5&exclude-non-failed-tests=&sort-by-flakiness=&dashboard=%s"

var reportFields log.Fields

// Tracks status of Jobs for a single TestGrid TabGroup
type TabGroupStatus struct {
	Name               string
	CollectedAt        time.Time
	Count              int
	TabGroupSummaryUrl string
	FlakingJobs        map[string]jobStatus
	PassingJobs        map[string]jobStatus
	FailedJobs         map[string]jobStatus
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

	jobsSummary := make(map[string]jobStatus)

	err = json.Unmarshal(body, &jobsSummary)
	if err != nil {
		t.logError("UnMarshalling reponse body", err)
		return err
	}

	t.FlakingJobs = make(map[string]jobStatus, 0)

	for name, job := range jobsSummary {
		if job.OverallStatus == "FLAKY" {
			t.FlakingJobs[name] = jobsSummary[name]
		}
	}

	t.FailedJobs = make(map[string]jobStatus, 0)
	for name, job := range jobsSummary {
		if strings.Compare(job.OverallStatus, "FAILED") == 0 {
			t.FailedJobs[name] = job
		}
	}

	t.PassingJobs = make(map[string]jobStatus, 0)
	for name, job := range jobsSummary {
		if strings.EqualFold(job.OverallStatus, "PASSING") {
			t.PassingJobs[name] = job
		}
	}

	t.Count = len(jobsSummary)
	return nil
}

func (t *TabGroupStatus) logError(action string, err error, fields ...string) log.Logger {
	var augmentedLogger = log.WithFields(reportFields).WithField(
		"ACTION", action,
	)
	for i, field := range fields {
		augmentedLogger.WithField(fmt.Sprintf("ExtraField %d",i),field)
	}
	augmentedLogger.Error("%v", err)

	return log.Logger{}
}

func main() {

	srInforming := &TabGroupStatus{
		Name:        "sig-release-master-informing",
		CollectedAt: time.Now(),
		TabGroupSummaryUrl: fmt.Sprintf(TG_TABGROUP_SUMMARY_FMT,
			"sig-release-master-informing"),
	}

	log.SetFormatter(&log.JSONFormatter{})
	reportFields = log.Fields{
		"DATA BEING RETRIEVED": "Job Status Summary TestGrid TabGroup",
		"TEST_GRID TAB_GROUP":  srInforming.Name,
		"COLLECTION TIME":      srInforming.CollectedAt,
		"TB GRP SMMRY URL":     srInforming.TabGroupSummaryUrl,
	}

	srInforming.CollectStatus()
	srInforming.CollectFlakyTests()

	for jobName, jobStatus := range srInforming.FlakingJobs {
		jobFlakyTests := jobStatus.jobTestResults
		for _, flakyTest := range jobFlakyTests.Tests {
			fmt.Printf("%s,%s,%s,\"%s\",\"%s\",%s\n",
				srInforming.CollectedAt.Format(time.UnixDate),
				jobStatus.OverallStatus, jobName, flakyTest.sig,
				flakyTest.Name, jobStatus.url)
		}
	}
}
