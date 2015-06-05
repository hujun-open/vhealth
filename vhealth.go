/*
vhealth, A tool to visualize Apple iOS health App exported XML data
read the XML file, and export HTML page based Chart
just invoke "vhealth -fname <xml-file-name>", and use a broswer to visit http://127.0.0.1:9090/
the listening port and address could be changed via command line arguments.

ver0.1
Hu Jun
June.4.2015

*/
package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//following are the struct of paring exported xml
type HealthData struct {
	XMLName xml.Name `xml:"HealthData"`
	Locale  string   `xml:"locale,attr"`
	Records []Record `xml:"Record"`
}

type Record struct {
	XMLName     xml.Name `xml:"Record"`
	Type        string   `xml:"type,attr"`
	Source      string   `xml:"source,attr"`
	Unit        string   `xml:"unit,attr"`
	StartDate   string   `xml:"startDate,attr"`
	EndDate     string   `xml:"endDate,attr"`
	Value       float64  `xml:"value,attr"`
	RecordCount int      `xml:"recordCount,attr"`
}

//end of parsing structs
type ExpHealthRecords struct {
	Filename        string
	ParsedData      HealthData
	TypeSet         map[string]bool
	YearSet         map[int]bool
	NumberOfRecords int
	LastestDate     string
}

func newExpHealthRecords(filename string) (result *ExpHealthRecords, err error) {
	result = nil
	err = nil
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	result = new(ExpHealthRecords)
	result.Filename = filename
	result.TypeSet = make(map[string]bool)
	result.YearSet = make(map[int]bool)
	result.NumberOfRecords = 0
	result.LastestDate = "0"
	err = xml.Unmarshal(data, &(result.ParsedData))
	if err != nil {
		return nil, err
	}
	result.getSummary()
	return
}

func (hrs ExpHealthRecords) getAllHourCountersForTheDay(rtype string,
	year string, month string, day string) (result map[int]float64, unit string) {
	//return all hour aggregated records in the given day in a map
	result = make(map[int]float64)
	for _, r := range hrs.ParsedData.Records {
		if r.Type == rtype && strings.HasPrefix(r.StartDate, year+month+day) {
			hour, err := strconv.Atoi(r.StartDate[8:10])
			unit = r.Unit
			if err == nil {
				_, ok := result[hour]
				if ok {
					result[hour] += r.Value
				} else {
					result[hour] = r.Value
				}
			}
		}
	}
	return
}

func (hrs ExpHealthRecords) getUserFriendlyTypeStr(apple_type string) (user_type string) {
	tmp_str := apple_type[24:]
	user_type = ""
	m := 0
	for n := 1; n < len(tmp_str); n++ {
		if string(tmp_str[n]) == strings.ToUpper(string(tmp_str[n])) {
			user_type += tmp_str[m:n] + " "
			m = n
		}
	}
	user_type += tmp_str[m:]
	return
}

func (hrs *ExpHealthRecords) getSummary() {
	hrs.NumberOfRecords = 0
	for _, r := range hrs.ParsedData.Records {
		hrs.NumberOfRecords += 1
		_, ok := hrs.TypeSet[r.Type]
		if !ok {
			hrs.TypeSet[r.Type] = true
		}
		r_year, err := strconv.Atoi(r.StartDate[0:4])
		if err == nil {
			_, ok := hrs.YearSet[r_year]
			if !ok {
				hrs.YearSet[r_year] = true
			}
		}
		ldate, err := strconv.Atoi(r.EndDate[0:8])
		if err == nil {
			current_latestdate, _ := strconv.Atoi(hrs.LastestDate)
			if ldate > current_latestdate {
				hrs.LastestDate = r.EndDate[0:8]
			}
		}
	}

}

func (hrs ExpHealthRecords) getAllMonthCountersForTheYear(rtype string,
	year string) (result map[int]float64, unit string) {
	//return all month aggregated records in the given year in a map
	result = make(map[int]float64)
	for _, r := range hrs.ParsedData.Records {
		if r.Type == rtype && strings.HasPrefix(r.StartDate, year) {
			month, err := strconv.Atoi(r.StartDate[4:6])
			unit = r.Unit
			if err == nil {
				_, ok := result[month]
				if ok {
					result[month] += r.Value
				} else {
					result[month] = r.Value
				}
			}
		}
	}
	return
}

func (hrs ExpHealthRecords) getAllDayCountersForTheMonth(rtype string,
	year string, month string) (result map[int]float64, unit string) {
	//return all day aggregated records in the given month in a map
	result = make(map[int]float64)

	for _, r := range hrs.ParsedData.Records {
		if r.Type == rtype && strings.HasPrefix(r.StartDate, year+month) {
			day, err := strconv.Atoi(r.StartDate[6:8])
			unit = r.Unit
			if err == nil {
				_, ok := result[day]
				if ok {
					result[day] += r.Value
				} else {
					result[day] = r.Value
				}
			}
		}
	}
	return
}

type WebServer struct {
	EHR           *ExpHealthRecords
	chartTemplate string
}

func (svr WebServer) returnJSFile(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	work_dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	jsfilename := "Chart.js"
	if err == nil {
		jsfilename = filepath.Join(work_dir, "Chart.js")
	}

	js, err := ioutil.ReadFile(jsfilename)
	if err == nil {
		w.Write([]byte(js))
	}

}

func (svr WebServer) showChart(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	html_str := ""
	switch req.Form["subutton"][0] {
	case "DayChart":
		counter_list, unit := svr.EHR.getAllHourCountersForTheDay(req.Form["rtype"][0], req.Form["year"][0], req.Form["month"][0], req.Form["day"][0])
		list_str := "["
		key_str := "["
		for n := 0; n <= 23; n++ {
			key_str += fmt.Sprintf("'hour %d',", n)
			v, ok := counter_list[n]
			if ok {
				list_str += fmt.Sprintf("%.2f,", v)
			} else {
				list_str += "0,"
			}
		}
		list_str += "]"
		key_str += "]"
		label_str := fmt.Sprintf("%[1]s, Day Chart, %[2]s-%[3]s-%[4]s", svr.EHR.getUserFriendlyTypeStr(req.Form["rtype"][0]), req.Form["year"][0], req.Form["month"][0], req.Form["day"][0])
		if len(counter_list) == 0 {
			label_str += "<br>Error: No Record!"
		}
		html_str = fmt.Sprintf(svr.chartTemplate, label_str, key_str, list_str, unit)

	case "MonthChart":
		counter_list, unit := svr.EHR.getAllDayCountersForTheMonth(req.Form["rtype"][0], req.Form["year"][0], req.Form["month"][0])
		list_str := "["
		key_str := "["
		for n := 1; n <= 31; n++ {
			key_str += fmt.Sprintf("'day %d',", n)
			v, ok := counter_list[n]
			if ok {
				list_str += fmt.Sprintf("%.2f,", v)
			} else {
				list_str += "0,"
			}
		}
		list_str += "]"
		key_str += "]"
		label_str := fmt.Sprintf("%[1]s, Month Chart, %[2]s-%[3]s", svr.EHR.getUserFriendlyTypeStr(req.Form["rtype"][0]), req.Form["year"][0], req.Form["month"][0])
		html_str = fmt.Sprintf(svr.chartTemplate, label_str, key_str, list_str, unit)

	case "YearChart":
		counter_list, unit := svr.EHR.getAllMonthCountersForTheYear(req.Form["rtype"][0], req.Form["year"][0])
		list_str := "["
		key_str := "["
		for n := 1; n <= 12; n++ {
			key_str += fmt.Sprintf("'month %d',", n)
			v, ok := counter_list[n]
			if ok {
				list_str += fmt.Sprintf("%.2f,", v)
			} else {
				list_str += "0,"
			}
		}
		list_str += "]"
		key_str += "]"
		label_str := fmt.Sprintf("%[1]s, Year Chart, %[2]s", svr.EHR.getUserFriendlyTypeStr(req.Form["rtype"][0]), req.Form["year"][0])
		html_str = fmt.Sprintf(svr.chartTemplate, label_str, key_str, list_str, unit)

	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html_str))

}

func (svr WebServer) Home(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	const template_str = `<!DOCTYPE html>
<html>
<head>
  <style>
  /* Elegant Aero */
  .elegant-aero {
      margin-left:auto;
      margin-right:auto;

      max-width: 700px;
      background: #D2E9FF;
      padding: 20px 20px 20px 20px;
      font: 12px Arial, Helvetica, sans-serif;
      color: #666;
  }
  .elegant-aero h1 {
      font: 24px "Trebuchet MS", Arial, Helvetica, sans-serif;
      padding: 10px 10px 10px 20px;
      display: block;
      background: #C0E1FF;
      border-bottom: 1px solid #B8DDFF;
      margin: -20px -20px 15px;
  }
  .elegant-aero h1>span {
      display: block;
      font-size: 11px;
  }

  .elegant-aero label>span {
      float: left;
      margin-top: 10px;
      color: #5E5E5E;
  }
  .elegant-aero label {
      display: block;
      margin: 0px 0px 5px;
  }
  .elegant-aero label>span {
      float: left;
      width: 20%%;
      text-align: right;
      padding-right: 15px;
      margin-top: 10px;
      font-weight: bold;
  }
  .elegant-aero input[type="text"], .elegant-aero input[type="email"], .elegant-aero textarea, .elegant-aero select {
      color: #888;
      width: 70%%;
      padding: 0px 0px 0px 5px;
      border: 1px solid #C5E2FF;
      background: #FBFBFB;
      outline: 0;
      -webkit-box-shadow:inset 0px 1px 6px #ECF3F5;
      box-shadow: inset 0px 1px 6px #ECF3F5;
      font: 200 12px/25px Arial, Helvetica, sans-serif;
      height: 30px;
      line-height:15px;
      margin: 2px 6px 16px 0px;
  }
  .elegant-aero textarea{
      height:100px;
      padding: 5px 0px 0px 5px;
      width: 70%%;
  }
  .elegant-aero select {
      background: #fbfbfb  no-repeat right;
      text-indent: 0.01px;
      text-overflow: '';
      width: 70%%;
  }
  .elegant-aero .button{
      padding: 10px 30px 10px 30px;
      background: #66C1E4;
      border: none;
      color: #FFF;
      box-shadow: 1px 1px 1px #4C6E91;
      -webkit-box-shadow: 1px 1px 1px #4C6E91;
      -moz-box-shadow: 1px 1px 1px #4C6E91;
      text-shadow: 1px 1px 1px #5079A3;

  }
  .elegant-aero .button:hover{
      background: #3EB1DD;
  }

  </style>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<title>iOS Health App Export XML Visualizer</title>
</head>
<body>
<form method="post" action="/action" class="elegant-aero">
<h1>
  Parsed File: %[1]s
</h1>
<center>
<span>
  Total %[2]d records
</span>
</center>
<label>
<span>Type :</span><select name="rtype">
  %[3]s
</select>
</label>
<label>
<span>Year :</span><select name="year">
%[4]s
</select>
</label>
<label>
<span>Moth :</span><select name="month">
%[5]s
</select>
</label>
<label>
<span>Day :</span><select name="day">
  %[6]s
</select>
</label>
<label>
   <span>&nbsp;</span>
   <button type="submit" class="button" value="DayChart" name="subutton" />Show Day Chart</button>
   <button type="submit" class="button" value="MonthChart" name="subutton" />Show Month Chart</button>
   <button type="submit" class="button" value="YearChart" name="subutton" />Show Year Chart</button>
</label>
<label>

</label>
</body>
</html>
`
	rtype_str := ""
	for t, _ := range svr.EHR.TypeSet {
		rtype_str += fmt.Sprintf(`<option value="%[1]s">%[2]s</option>`+"\n", t, svr.EHR.getUserFriendlyTypeStr(t))
	}
	year_str := ""
	for y, _ := range svr.EHR.YearSet {
		ys, _ := strconv.Atoi(svr.EHR.LastestDate[0:4])
		slcts := ""
		if y == ys {
			slcts = "selected"
		}
		year_str += fmt.Sprintf(`<option value="%[1]d" %[2]s>%[1]d</option>`+"\n", y, slcts)
	}
	month_str := ""
	for m := time.January; m <= time.December; m++ {
		slcts := ""
		lmonth, _ := strconv.Atoi(svr.EHR.LastestDate[4:6])
		if lmonth == int(m) {
			slcts = "selected"
		}
		month_str += fmt.Sprintf(`<option value="%02[1]d" %[2]s>%[3]s</option>`+"\n", m, slcts, m.String())
	}

	day_str := ""
	for d := 1; d <= 31; d++ {
		slcts := ""
		lday, _ := strconv.Atoi(svr.EHR.LastestDate[6:8])
		if lday == d {
			slcts = "selected"
		}
		day_str += fmt.Sprintf(`<option value="%02[1]d" %[2]s>%[3]d</option>`+"\n", d, slcts, d)
	}

	result_htmls := fmt.Sprintf(template_str, svr.EHR.Filename, svr.EHR.NumberOfRecords, rtype_str, year_str, month_str, day_str)
	w.Write([]byte(result_htmls))
}

func newWebServer(filename string) (result *WebServer, err error) {
	result = new(WebServer)
	result.EHR, err = newExpHealthRecords(filename)
	if err != nil {
		return nil, err
	}
	result.chartTemplate = `<!doctype html>
<html>
	<head>
		<title>iOS Health Chart</title>
		<script src="Chart.js"></script>
	</head>
	<body>
    <h1>%[1]s</h1>
	<h3>Unit: %[4]s</h3>
		<div style="width:50%%">
			<div>
				<canvas id="canvas" height="500" width="1000"></canvas>
			</div>
		</div>

	<script>
		var lineChartData = {
			labels : %[2]s,
			datasets : [
				{
					label: "My Second dataset",
					fillColor : "rgba(151,187,205,0.2)",
					strokeColor : "rgba(151,187,205,1)",
					pointColor : "rgba(151,187,205,1)",
					pointStrokeColor : "#fff",
					pointHighlightFill : "#fff",
					pointHighlightStroke : "rgba(151,187,205,1)",
		          	data : %[3]s
                }
			]

		}

	window.onload = function(){
		var ctx = document.getElementById("canvas").getContext("2d");
		window.myLine = new Chart(ctx).Line(lineChartData, {
			responsive: true,

		});
	}
	</script>
	</body>
</html>`
	http.HandleFunc("/action", result.showChart)
	http.HandleFunc("/", result.Home)
	http.HandleFunc("/Chart.js", result.returnJSFile)
	return
}

func main() {
	version_str := "Visualizer tool v0.1 for Apple iOS Health App exported XML\nby Hu Jun 2015.June\n-help for usage\n\n"
	fmt.Println(version_str)
	var port = flag.Int("port", 9090, "specify the listing port")
	var ip = flag.String("ip", "127.0.0.1", "specify listening ip address")
	var filename = flag.String("fname", "", "specify the file name of the exported XML")
	flag.Parse()
	if *filename == "" {
		fmt.Println("Error: Miss file name of XML!")
		flag.PrintDefaults()
		return
	}
	myhr, err := newWebServer(*filename)
	if err != nil {
		fmt.Println(myhr)
		log.Fatal(err)
	}
	fmt.Printf("Starting HTTP server @ %s:%d\n", *ip, *port)
	fmt.Printf("Use a web broswer to visit http://%s:%d/\n", *ip, *port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", *ip, *port), nil)
	if err != nil {
		log.Fatal(err)
	}

}
