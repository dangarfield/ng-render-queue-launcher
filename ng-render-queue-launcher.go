package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/mitchellh/go-ps"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Result struct {
	Scene    string `json: "scene"`
	Camera   string `json: "camera"`
	Render   string `json: "render"`
	Status   string `json: "status"`
	Time     string `json: "time"`
	Duration string `json: "duration"`
}
type Config struct {
	Mode          string   `json: "mode"`
	Scenes        []string `json: "scenes"`
	Results       []Result `json: "results"`
	Installed     bool     `json: "installed"`
	Validated     bool     `json: "validated"`
	ShouldTurnOff bool     `json: "shouldTurnOff"`
	ConfigPath    string   `json: "configPath"`
	DazPath       string   `json: "dazPath"`
	LauncherPath  string   `json: "launcherPath"`
}

func openConfig() Config {
	jsonPath := path.Join(os.Getenv("APPDATA"), "DAZ 3D", "Studio4", "scripts", "Render Queue", "render-queue-data.json")
	log.Println("jsonPath", jsonPath)
	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		log.Println(err)
	}
	log.Println("Successfully Opened users.json")
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var config Config

	json.Unmarshal(byteValue, &config)

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	return config
}
func isDazRunning() bool {
	p, err := ps.Processes()
	if err != nil {
		log.Fatalf("err: %s", err)
	}
	// log.Println(p)
	running := false
	for _, s := range p {
		// log.Println(i, s.Executable(), s.Pid(), s.PPid())
		if s.Executable() == "DAZStudio.exe" {
			running = true
		}
	}
	return running
}
func executionLoop(progress *widget.Label) {
	log.Println("-----------------------------------------------------")
	log.Println("------------------- ExecutionLoop -------------------")
	log.Println("-----------------------------------------------------")

	config := openConfig()
	// - mode != render - exit launcher app
	if config.Mode == "configaaaa" {
		log.Println("Not in render mode, exiting")
		progress.SetText("Not in render mode, exiting...")
		os.Exit(0)
	}
	dazRunning := isDazRunning()
	log.Println("Config", config)
	log.Println("DAZ running", dazRunning)

	// - shouldRender true && daz process is running - continue wait
	if dazRunning {
		log.Println("DAZ is running, continue waiting")
		progress.SetText("DAZ is running, continue waiting...")
		return
	}

	log.Println("DAZ is not running, what's next?")
	// - shouldRender true && daz process is not running && config.scenes.length > 0 - launch daz -> render mode
	// - shouldRender true && daz process is not running && config.scenes.length === 0 && shouldTurnoff false - launch daz -> results mode
	if !config.ShouldTurnOff {
		log.Println("Launch DAZ for rendering or results")
		cmd := exec.Command(config.DazPath)
		cmd.Start()
		log.Println("DAZ Launched")
		progress.SetText("DAZ Launched..")
		return
	} // - shouldRender true && daz process is not running && config.scenes.length === 0 && shouldTurnoff false - turn off computer
	if config.ShouldTurnOff {
		log.Println("Turning off computer")
		progress.SetText("Turning off computer...")
		if err := exec.Command("cmd", "/C", "shutdown", "/s").Run(); err != nil {
			log.Println("Failed to initiate shutdown:", err)
		}

		return
	}

	log.Println("Should not see this")
}
func setupGUI() (fyne.Window, *widget.Label) {
	a := app.New()
	w := a.NewWindow("NG Render Queue")

	progress := widget.NewLabel("Loading...")
	image := canvas.NewImageFromFile("Icon.png")
	image.FillMode = canvas.ImageFillOriginal
	w.SetContent(container.NewVBox(
		image,
		progress,
	))
	return w, progress
}
func exeLoop(progress *widget.Label) {
	for true {
		executionLoop(progress)
		time.Sleep(5 * time.Second)
	}
}
func main() {
	log.Println("START")
	w, progress := setupGUI()
	go exeLoop(progress)
	w.ShowAndRun()

	log.Println("END")

}

// go build .\ng-render-queue-launcher.go
// fyne package -os windows -icon .\Icon.png
