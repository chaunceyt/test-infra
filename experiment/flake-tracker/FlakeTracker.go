package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const TG_TABGROUP_SUMMARY_FMT string = "https://testgrid.k8s.io/%s/summary"

type jobStatus struct {
	OverallStatus           string `json:"overall_status"`
	Alert                   string `json:"alert"`
	LastRun                 int64  `json:"last_run_timestamp"`    // 1601294497000,
	LastUpdate              int64  `json:"last_update_timestamp"` // 1601374005,
	LatestGreenRun          string `json:"latest_green"`          // "",
	LatestStatusIcon        string `json:"overall_status_icon"`   // "warning",
	LatestStatusDescription string `json:"status"` // "2 of 10 (20.0%) recent columns passed (1 0867 of 10893 or 99.8% cells)",

}

// Tracks status of Jobs for a single TestGrid TagGroup
type TabGroupStatus struct {
	Name string
	CollectedAt time.Time
	Count int
	FlakingJobs map[string] jobStatus
	PassingJobs map[string] jobStatus
	FailedJobs map[string] jobStatus
}

// CollectStatus populates t with job status summary data from TestGrid
func (t *TabGroupStatus) CollectStatus() error {
	url := fmt.Sprintf(TG_TABGROUP_SUMMARY_FMT, t.Name)

	resp, err := http.Get(url)

	if err != nil { // TODO convert to use logger
		fmt.Printf("%v", err)
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { // TODO convert to use logger
		fmt.Printf("%v", err)
		return err
	}

	jobsSummary := make(map[string]jobStatus)
	err = json.Unmarshal(body, &jobsSummary)
	if err != nil {
		return err
	}
	t.FlakingJobs = make(map[string]jobStatus, 0)
	for name, job := range jobsSummary {
		if strings.EqualFold(job.OverallStatus,"FLAKY") {
			t.FlakingJobs[name] = job
		}
	}

	t.FailedJobs = make(map[string]jobStatus, 0)
	for name, job := range jobsSummary {
		if strings.EqualFold(job.OverallStatus,"FAILED") {
			t.FailedJobs[name] = job
		}
	}

	t.PassingJobs = make(map[string]jobStatus, 0)
	for name, job := range jobsSummary {
		if strings.EqualFold(job.OverallStatus,"PASSING") {
			t.PassingJobs[name] = job
		}
	}
	t.Count = len(jobsSummary)
	return nil
}

func main() {
	tgs := &TabGroupStatus{Name: "sig-release-master-blocking", CollectedAt: time.Now()}
	tgs.CollectStatus()
	fmt.Printf("%+v\n",tgs)
}
