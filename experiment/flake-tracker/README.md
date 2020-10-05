# Flake Trackeri WIP#
CURRENTLY IMPLEMENTED -  Creates a point-in-time CSV report listing flakying tests found on the Jobs reported as FLAKY in the TestGridSummary for sig-release-master-blocking

TODO - For a given TestGrid TabGroup and a coresponding Github Project Board set up to monitor deflaking work 
the Flake Tracker provides a report on the status of jobs with respect to flakiness of TabGroups CI Jobs 

TODO - For every currently flaking job, the report retrieves 
 the names of the flaky tests and then goes off to the Project Board to check for the presence of a logged issue and retireves the issues and their status from the board.

This report is designed to support the work of any team who is interested in detecting, triaging and fixing flakes on the Kubernetes project.

The report offers the following benefits :

  * helps prevent logging duplicate issues for a test that is producing non-deterministic results.

  * TODO provides a flake detection percentage 

  * shows distribution of flakes accross the project per job, per SIG

  * TODO shows distribution of categorised effort (Awaiting response, triaged, PR submitted, monitoring, fixed) accross the project per job, per sig

# Build#
The Project is built using go version go.1.14.4

``` 
go build && ./flake-tracker
```

# Usage#
Run a report on sig-release-master-blocking

``` 
./flake-tracker 
```

## Parameters and environment ##
At present no parameters are required to run the program future versions may have the following cmd line flags
TODO 
--config file YAML file that contains report configuration
--gh-token GitHub Oauth2 token
--tab-group TestGrid TabGroup
--project-board GithubProjectBoard yaml
--output Output format - json, csv, org
--port - if specificed starts a server listenting on port and displays a HTML version of the report
