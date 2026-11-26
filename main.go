package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/egoist/mygo"
)

const version = "0.1.0"

func main() {
	capture := flag.String("capture", "", "capture the first frame to this PNG and exit")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("FileSpace233", version)
		return
	}

	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	app := NewApp()
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnSecondInstance(func([]string, string) { app.showWindow() })
	mygo.App.WhenReady(func() {
		mygo.App.SetLocale("zh-Hans")
		app.start(*capture)
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
