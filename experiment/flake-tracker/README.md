# Flake Tracker

Creates a point-in-time CSV (for now) report listing tests that produce non-determinstic results (NDRs) found on the Jobs reported as FLAKY in the TestGridSummary for sig-release-blocking and sig-release-informing

The report offers the following benefits :

  * provides automatic on-demand status updates for weekly Release Team Meeting
  
  * shows distribution of NDRs accross the project per job, per SIG 
  
  * TODO shows what NDRs are and are not being tracked by GH Issues 

  * TODO shows distribution of categorised effort (Awaiting response, triaged, PR submitted, monitoring, fixed) accross the project per job, per sig

# Build#
The Project is built using go version go.1.14.4

``` 
go build && ./flake-tracker
```

# Usage#
Run a report on sig-release-blocking

``` 
./flake-tracker 
```

## Parameters and environment ##
At present no parameters are required to run the program future versions may have the following cmd line flags
TODO 
* --config file YAML file that contains report configuration, tabgroups, project boards, output format, datastore
* --gh-token / env var GitHub Oauth2 token
* --tab-group TestGrid TabGroup
* --project-board GithubProjectBoard yaml
* --output Output format - json, csv, org
* --port - if specificed starts a server listenting on port and displays a HTML version of the report
