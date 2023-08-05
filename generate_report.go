package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/goodsign/monday"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Data struct {
	TestTime time.Time
	Result   OutputTest
}

func GetData() []Data {
	db, err := leveldb.OpenFile("speeds.db", nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	iter := db.NewIterator(nil, nil)

	var results []Data

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		dataRes := &OutputTest{}

		if err := json.Unmarshal(value, dataRes); err != nil {
			panic(err)
		}

		var tm time.Time

		if err := tm.UnmarshalBinary(key); err != nil {
			panic(err)
		}

		result := &Data{
			TestTime: tm,
			Result:   *dataRes,
		}

		results = append(results, *result)
	}

	return results
}

type ReportData struct {
	TestTime      time.Time
	Server        string
	ServerLoc     string
	Ping          time.Duration
	Jitter        time.Duration
	DownloadSpeed int
	UploadSpeed   int
}

func GenTable(month time.Month, year int) (string, string) {
	funcMap := template.FuncMap{
		"humBytes": func(i int) string {
			return humanize.IBytes(uint64(i * 100))
		},
	}

	p := filepath.Join("templates", "table.html")
	tmpl := template.Must(template.New("table.html").Funcs(funcMap).ParseFiles(p))

	data := GetData()
	var reportData []ReportData

	endDate := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	initialDate := endDate.AddDate(0, -1, 0)

	for _, s := range data {
		for _, t := range s.Result.Resultados {
			reportData = append(reportData, ReportData{
				TestTime:      s.TestTime,
				Server:        t.Sponsor,
				ServerLoc:     fmt.Sprintf("[%s, %s]", t.Lat, t.Lon),
				Ping:          RoundTime(t.Latency, 0),
				Jitter:        RoundTime(t.Jitter, 0),
				DownloadSpeed: int(math.Round(t.DLSpeed)),
				UploadSpeed:   int(math.Round(t.ULSpeed)),
			})
		}
	}

	reportData = Filter(reportData, func(n ReportData) bool {
		return n.TestTime.After(initialDate) && n.TestTime.Before(endDate)
	})

	var doc bytes.Buffer
	err := tmpl.ExecuteTemplate(&doc, "table.html", reportData)
	if err != nil {
		panic(err)
	}

	sourceConn := data[0].Result.Provedor.String()

	return doc.String(), sourceConn
}

func GenMedian(month time.Month, year int) (int, int, int) {
	data := GetData()
	var reportData []ReportData

	for _, s := range data {
		for _, t := range s.Result.Resultados {
			reportData = append(reportData, ReportData{
				TestTime:      s.TestTime,
				Ping:          t.Latency,
				DownloadSpeed: int(math.Round(t.DLSpeed)),
				UploadSpeed:   int(math.Round(t.ULSpeed)),
			})
		}
	}

	endDate := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	initialDate := endDate.AddDate(0, -1, 0)

	reportData = Filter(reportData, func(n ReportData) bool {
		return n.TestTime.After(initialDate) && n.TestTime.Before(endDate)
	})

	mappedPing := Map(reportData, func(n ReportData) int64 {
		return int64(n.Ping / time.Millisecond)
	})

	pingTotal := Reduce(mappedPing, func(acc, current int64) int64 {
		return acc + current
	}, int64(0))

	pingMedian := int(pingTotal / int64(len(reportData)))

	mappedDownloadSpeed := Map(reportData, func(n ReportData) int {
		return n.DownloadSpeed
	})

	downloadSpeedTotal := Reduce(mappedDownloadSpeed, func(acc, current int) int {
		return acc + current
	}, 0)

	downloadSpeedMedian := (downloadSpeedTotal / len(reportData))

	mappedUploadSpeed := Map(reportData, func(n ReportData) int {
		return n.UploadSpeed
	})

	uploadSpeedTotal := Reduce(mappedUploadSpeed, func(acc, current int) int {
		return acc + current
	}, 0)

	uploadSpeedMedian := (uploadSpeedTotal / len(reportData))

	return pingMedian, downloadSpeedMedian, uploadSpeedMedian
}

func GenReport(monthYear time.Time) {
	/* graph := chart.Chart{
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: []float64{1.0, 2.0, 3.0, 4.0},
				YValues: []float64{1.0, 2.0, 3.0, 4.0},
			},
		},
	}

	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		panic(err)
	}

	encodedText := base64.StdEncoding.EncodeToString(buffer.Bytes()) */

	caser := cases.Title(language.BrazilianPortuguese)
	montYer := caser.String(monday.Format(monthYear, "January/2006", monday.LocalePtBR))

	table, sourceConn := GenTable(monthYear.Month(), monthYear.Year())
	ping, down, upl := GenMedian(monthYear.Month(), monthYear.Year())

	html := fmt.Sprintf(`<meta http-equiv="Content-Type" content="text/html; charset=UTF-8"><html><center><h1>Relatório %s</h1></center><br>Origem: %s<br><br>%s<br><b>Média Ping: %dms | Média Download: %d Mbps | Média Upload: %d Mbps</b></html>`,
		montYer, sourceConn, table, ping, down, upl)

	htmlFile, err := os.Create("report.html")
	if err != nil {
		panic(err)
	}

	_, err = htmlFile.WriteString(html)
	if err != nil {
		panic(err)
	}
}
