# Flake Tracker#
For a given TestGrid TabGroup and a coresponding Github Project Board set up to monitor deflaking work 
the Flake Tracker provides a report on the status of jobs with respect to flakiness of TabGroups CI Jobs 

For every currently flaking job, the report retrieves 
 the names of the flaky tests and then goes off to the Project Board to check for the presence of a logged issue and retireves the issues and their status from the board.

This report is designed to support the work of any team who is interested in detecting, triaging and fixing flakes on the Kubernetes project.

The report offers the following benefits :  

  * prevents people from logging duplicate issues for a test that is producing non-deterministic results.

  * provides a flake detection percentage 

  * shows distribution of flakes accross the project per job, per SIG

  * shows distribution of categorised effort (Awaiting response, triaged, PR submitted, monitoring, fixed) accross the project per job, per sig
