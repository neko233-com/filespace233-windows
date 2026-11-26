package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/egoist/mygo"
)

const version = "0.3.0"

func main() {
	capture := flag.String("capture", "", "capture the first frame to this PNG and exit")
	captureQuad := flag.Bool("capture-quad", false, "capture the 2x2 layout")
	captureInspector := flag.Bool("capture-inspector", false, "capture with the inspector open")
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
		app.start(*capture, *captureQuad, *captureInspector)
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
