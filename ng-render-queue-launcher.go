package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/shirou/gopsutil/v3/process"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Scene struct {
	Path   string `json: "path"`
	Status string `json: "status"`
}
type Result struct {
	Scene    string `json: "scene"`
	Camera   string `json: "camera"`
	Render   string `json: "render"`
	Status   string `json: "status"`
	Time     string `json: "time"`
	Duration string `json: "duration"`
}
type Config struct {
	Error         bool     `json: "error"` // Not in json, just for error handling
	Mode          string   `json: "mode"`
	Scenes        []Scene  `json: "scenes"`
	Results       []Result `json: "results"`
	Installed     bool     `json: "installed"`
	ShouldTurnOff bool     `json: "shouldTurnOff"`
	ConfigPath    string   `json: "configPath"`
	DazPath       string   `json: "dazPath"`
	LauncherPath  string   `json: "launcherPath"`
}

func openConfig() Config {
	jsonPath := path.Join(os.Getenv("APPDATA"), "DAZ 3D", "Studio4", "scripts", "NG Render Queue", "ng-render-queue-data.json")
	log.Println("jsonPath", jsonPath)
	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		log.Println(err)
		badConfig := new(Config)
		badConfig.Error = true
		return *badConfig
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
	processes, err := process.Processes()
	if err != nil {
		//log.Fatalf("err: %s", err)
	}
	running := false
	for _, p := range processes {
		n, err := p.Name()
		if err != nil {
			//log.Printf("err: %s", err)
		} else {
			//log.Println("Process: ", n)
			if n == "DAZStudio.exe" {
				//log.Println("  KILL Process: ", n)
				running = true
			}
		}
	}
	return running
}
func forceCloseDaz() {
	processes, err := process.Processes()
	if err != nil {
		//log.Fatalf("err: %s", err)
	}
	for _, p := range processes {
		n, err := p.Name()
		if err != nil {
			//log.Printf("err: %s", err)
		} else {
			//log.Println("Process: ", n)
			if n == "DAZStudio.exe" {
				//log.Println("  KILL Process: ", n)
				p.Kill()
			}
		}
	}

}

const closeCountDownInitial = 5

var closeCountDown = closeCountDownInitial

func executionLoop(progress *widget.Label) {
	log.Println("-----------------------------------------------------")
	log.Println("------------------- ExecutionLoop -------------------")
	log.Println("-----------------------------------------------------")

	config := openConfig()
	// - mode != render - exit launcher app

	if config.Error == true {
		log.Println("Bad config")
		progress.SetText("Unable to load config...")
		return
	}

	if config.Mode == "switchoff" {
		log.Println("Turning off computer")
		progress.SetText("Turning off computer...")
		if err := exec.Command("cmd", "/C", "shutdown", "/s").Run(); err != nil {
			log.Println("Failed to initiate shutdown:", err)
			progress.SetText("Failed to initiate computer shutdown")
		}
		return //os.Exit(0)
	}
	if config.Mode == "results" {
		log.Println("Results mode, exiting", config.Mode)
		progress.SetText("Results mode, exiting...")
		os.Exit(0)
	}
	if config.Mode != "render" && config.Mode != "closing" {
		log.Println("Not in render mode, exiting", config.Mode)
		progress.SetText("Not in render mode...")
		return //os.Exit(0)
	}

	dazRunning := isDazRunning()
	log.Println("Config", config)
	log.Println("DAZ running", dazRunning)

	// - shouldRender true && daz process is running - continue wait
	if dazRunning {
		log.Println("DAZ is running, continue waiting", config.Mode, closeCountDown)
		progress.SetText("DAZ is running, continue waiting...")
		if config.Mode == "closing" {
			progress.SetText("DAZ is closing, continue waiting... " + fmt.Sprint(closeCountDown))
			closeCountDown--
			if closeCountDown < 0 {
				progress.SetText("Force quiting DAZ")
				forceCloseDaz()
			}
		}
		return
	}
	closeCountDown = closeCountDownInitial

	log.Println("DAZ is not running, what's next?")
	// - shouldRender true && daz process is not running && config.scenes.length > 0 - launch daz -> render mode
	// - shouldRender true && daz process is not running && config.scenes.length === 0 && shouldTurnoff false - launch daz -> results mode

	var completeCount = 0
	for _, s := range config.Scenes {
		if s.Status == "Complete" {
			completeCount++
		}
	}

	log.Println("Scene length", len(config.Scenes) > 0)
	log.Println("Scene only has completes", len(config.Scenes) == completeCount)
	log.Println("Config set to turn off", config.ShouldTurnOff)

	if len(config.Scenes) > 0 && len(config.Scenes) == completeCount && config.ShouldTurnOff {
		log.Println("Turning off computer")
		progress.SetText("Turning off computer...")
		forceCloseDaz()
		if err := exec.Command("cmd", "/C", "shutdown", "/s").Run(); err != nil {
			log.Println("Failed to initiate shutdown:", err)
		}
		return
	} else {
		log.Println("Launch DAZ for rendering or results")
		// cmd := exec.Command(config.DazPath, "-scriptArg", config.ScriptPath)
		cmd := exec.Command(config.DazPath)
		cmd.Start()
		log.Println("DAZ Launched")
		progress.SetText("DAZ Launched... " + config.DazPath)
		return
	} // - shouldRender true && daz process is not running && config.scenes.length === 0 && shouldTurnoff false - turn off computer
}
func setupGUI() (fyne.Window, *widget.Label) {
	a := app.New()
	w := a.NewWindow("NG Render Queue")

	progress := widget.NewLabel("Loading...")

	imgResource, _ := fyne.LoadResourceFromURLString("https://rawcdn.githack.com/dangarfield/ng-render-queue-launcher/150626d9b30320edd54f2afc75e707ac10b26426/Icon.png")
	image := canvas.NewImageFromResource(imgResource)
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
