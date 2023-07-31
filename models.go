package main

import "github.com/showwin/speedtest-go/speedtest"

type OutputTest struct {
	Provedor   *speedtest.User    `json:"provedor"`
	Resultados []speedtest.Server `json:"resultados"`
}
