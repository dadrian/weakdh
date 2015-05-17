package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/dadrian/weakdh/check/handlers"
	"github.com/gin-gonic/gin"
)

type args struct {
	listenAddress    string
	logFileName      string
	outputFileName   string
	metadataFileName string
	mode             string
}

type output struct {
	outputFile   *os.File
	logFile      *os.File
	metadataFile *os.File
}

var a args
var o output

func openOrDefault(filename string, defaultFile *os.File) (*os.File, error) {
	if filename == "-" {
		return defaultFile, nil
	}
	return os.Create(filename)
}

func init() {
	flag.StringVar(&a.listenAddress, "listen-address", "127.0.0.1:8080", "ip:port to listen on")
	flag.StringVar(&a.outputFileName, "output-file", "-", "file to write data to")
	flag.StringVar(&a.mode, "mode", "debug", "debug|test|release")
	flag.Parse()

	gin.SetMode(a.mode)

	var err error

	if o.outputFile, err = openOrDefault(a.outputFileName, os.Stdout); err != nil {
		//zlog.Fatalf("Could not open output file %s: %s", a.outputFileName, err.Error())
		panic("fuck")
	}

}

func main() {
	r := gin.Default()

	checkGroup := r.Group("/check")
	handlers.UseServerCheck(checkGroup, o.outputFile)

	s := &http.Server{
		Addr:           a.listenAddress,
		Handler:        r,
		ReadTimeout:    12 * time.Second,
		WriteTimeout:   12 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}
