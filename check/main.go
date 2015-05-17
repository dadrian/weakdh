package main

import (
	"flag"
	"os"

	"github.com/dadrian/weakdh/check/handlers"
	"github.com/gin-gonic/gin"
)

type args struct {
	listenAddress    string
	logFileName      string
	outputFileName   string
	metadataFileName string
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
	flag.StringVar(&a.logFileName, "log-file", "-", "file to log errors to")

	var err error

	if o.outputFile, err = openOrDefault(a.outputFileName, os.Stdout); err != nil {
		//zlog.Fatalf("Could not open output file %s: %s", a.outputFileName, err.Error())
		panic("fuck")
	}

	if o.logFile, err = openOrDefault(a.logFileName, os.Stderr); err != nil {
		//	zlog.Fatalf("Could not open log file %s: %s", a.logFileName, err.Error())
		panic("fuck2")
	}

}

func main() {
	r := gin.Default()
	checkGroup := r.Group("/check")
	handlers.UseServerCheck(checkGroup)
	r.Run(a.listenAddress)
}
