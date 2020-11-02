package cmd

const reportTemplate = `<!-- DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 3.2 Final//EN" -->
<!DOCTYPE html>
<html lang="en">
<head>
  <title>{{.PageTitle}}</title>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.4.0/css/bootstrap.min.css">
  <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.4.1/jquery.min.js"></script>
  <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.4.0/js/bootstrap.min.js"></script>
</head>
<body>
  <nav class="navbar navbar-default">
    <div class="container-fluid">
      <div class="navbar-header">
        <a class="navbar-brand" href="#">Flake Tracker</a>
      </div>
      <ul class="nav navbar-nav">
        <li><a href="/">Home</a></li>
        <li><a href="/master-blocking">Browse by Master Blocking</a></li>
        <li><a href="/master-informing">Browse by Master Informing</a></li>
        <li><a href="/generate-report-files">Generate report files</a></li>
      </ul>
    </div>
  </nav>
<div class="container-fluid">
 <h1>{{.PageTitle}}</h1>
   <h2>Filterable Table</h2>
   <input class="form-control" id="reportInput" type="text" placeholder="Search..">
   <br>
<div class="row">
    <table class="table">
        <thead>
          <tr>
            <th scope="col">Collected</th>
            <th scope="col">JobName</th>
            <th scope="col">OwnerName</th>
            <th scope="col">Status</th>
            <th scope="col">TestName</th>
            <th scope="col">TestURL</th>
          </tr>
        </thead>
        <tbody id="reportTable">
        {{range .ReportItems}}
          <tr>
            <th scope="row">{{.Collected}}</th>
            <td>{{.JobName}}</td>
            <td>{{.OwnerName}}</td>
            <td>{{.Status}}</td>
            <td>{{.TestName}}</td>
            <td><a href="{{.TestURL}}">dashboard</a></td>
          </tr>
        {{end}}
        </tbody>
      </table>
    </div>
</div>
    
</div>
 <script>
  $(document).ready(function(){
    $("#reportInput").on("keyup", function() {
      var value = $(this).val().toLowerCase();
      $("#reportTable tr").filter(function() {
        $(this).toggle($(this).text().toLowerCase().indexOf(value) > -1)
      });
    });
  });
  </script>

</body>
</html>
`

const indexTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
  <title>{{.PageTitle}}</title>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.4.0/css/bootstrap.min.css">
  <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.4.1/jquery.min.js"></script>
  <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.4.0/js/bootstrap.min.js"></script>
</head>
<body>
  <nav class="navbar navbar-default">
    <div class="container-fluid">
      <div class="navbar-header">
        <a class="navbar-brand" href="#">Flake Tracker</a>
      </div>
      <ul class="nav navbar-nav">
        <li class="active"><a href="#">Home</a></li>
        <li><a href="/master-blocking">Browse by Master Blocking</a></li>
        <li><a href="/master-informing">Browse by Master Informing</a></li>
        <li><a href="/generate-report-files">Generate report files</a></li>
      </ul>
    </div>
  </nav>
<div class="container-fluid">
  <h3>{{.PageTitle}}</h3>
  <p>Current View of jobs reported as FLAKY in the TestGridSummary for sig-release-blocking and sig-release-informing"</p>
</div>

</body>
</html> 
`
