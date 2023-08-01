package main

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/goodsign/monday"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func RunSpeedTest() {
	var speedtestClient = speedtest.New()

	user, _ := speedtestClient.FetchUserInfo()

	serverList, _ := speedtestClient.FetchServers()
	targets, _ := serverList.FindServer([]int{})

	var results []speedtest.Server

	for _, s := range targets {
		s.PingTest(nil)
		s.DownloadTest()
		s.UploadTest()
		results = append(results, *s)
	}

	result := &OutputTest{
		Provedor:   user,
		Resultados: results,
	}

	jsonStr, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	db, err := leveldb.OpenFile("speeds.db", nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	encoding, err := time.Now().MarshalBinary()
	if err != nil {
		panic(err)
	}

	err = db.Put(encoding, jsonStr, nil)
	if err != nil {
		panic(err)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "SpeedTest"

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "test",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "report",
			Value: false,
		},
	}

	app.Action = func(c *cli.Context) error {
		if c.IsSet("test") && c.IsSet("report") {
			panic(errors.New("not valid two flags"))
		}

		if c.Bool("test") {
			RunSpeedTest()
		} else if c.Bool("report") {
			caser := cases.Title(language.BrazilianPortuguese)
			GenReport(caser.String(monday.Format(time.Now(), "January/2006", monday.LocalePtBR)))
		} else {
			panic(errors.New("need one flag"))
		}
		return nil
	}

	app.Run(os.Args)
}
