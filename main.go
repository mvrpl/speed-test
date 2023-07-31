package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/goodsign/monday"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/syndtr/goleveldb/leveldb"
)

func main() {
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

	db.Close()

	GenReport(strings.Title(monday.Format(time.Now(), "January/2006", monday.LocalePtBR)))
}
