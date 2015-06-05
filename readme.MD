# vhealth

This is a visualizer tool for Apple iOS Health App exported XML. written in GO.

This tool read in the XML file, and extract the records and render them in HTML/JS based chart.

## Export Data from App
1. open Health App
2. in "Health Data" tab, select "All"
3. Select export icon on up right corner of the screen
4. Enters the email address, where the zipped XML data will be sent to.

## Usage
```
vhealth -fname <xml-file-name>
```
Above command read the XML file, and start a HTTP server on http://127.0.0.1:9090. Use a web browser to visit the URL.

The listening address/port could be changed via CLI argument.
```
Usage of vhealth.exe:
  -fname="": specify the file name of the exported XML
  -ip="127.0.0.1": specify listening ip address
  -port=9090: specify the listing port
```
## Credit
This software uses following libs:
* Chart.js http://www.chartjs.org/
* Elegant Aero CSS HTML Form: http://www.sanwebe.com/2013/10/css-html-form-styles


## License
MIT
