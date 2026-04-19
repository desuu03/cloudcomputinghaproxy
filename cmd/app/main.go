package main

import (
	"App_Servidor_Imagenes/pkg/images"
	"App_Servidor_Imagenes/pkg/monitor"
	"App_Servidor_Imagenes/pkg/orchestrator"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	_ "embed"
)

type PageData struct {
	Images      []images.ImageInfo
	Hostname    string
	CourseName  string
	StudentName string
	CPUUsage   float64
	StressLevel int
	UpperThreshold float64
	LowerThreshold float64
	Interval    int
}

type Response struct {
	Status string `json:"status"`
}

var (
	stressLevel    int
	orch        *orchestrator.Orchestrator
	stressProc  *exec.Cmd
)

func main() {
	port := 8000
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}

	hostname, _ := os.Hostname()

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	templatePath := filepath.Join(exeDir, "templates", "index.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		templatePath = "templates/index.html"
	}

	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"safeURL": func(u string) template.URL {
			return template.URL(u)
		},
	}).ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("Error loading template: %v", err)
	}

	orch = orchestrator.Init("127.0.0.1", "/etc/haproxy/haproxy.cfg")

	go monitor.StartMonitoring(
		func() { scaleUp() },
		func() { scaleDown() },
	)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		imageDir := filepath.Join(exeDir, "ImagenesApp")
		if _, err := os.Stat(imageDir); os.IsNotExist(err) {
			imageDir = "ImagenesApp"
		}

		imageInfos, err := images.GetRandomBase64Images(imageDir, 3)
		if err != nil {
			http.Error(w, "Error loading images", http.StatusInternalServerError)
			return
		}

		cfg := monitor.GetConfig()

		data := PageData{
			Images:      imageInfos,
			Hostname:    hostname,
			CourseName: "Cloud Computing",
			StudentName: "David Alfonso Posso Cano & Santiago Navarro",
			CPUUsage:   monitor.GetUsage(),
			StressLevel: stressLevel,
			UpperThreshold: cfg.UpperThreshold,
			LowerThreshold: cfg.LowerThreshold,
			Interval:    int(cfg.Interval.Seconds()),
		}

		tmpl.Execute(w, data)
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			CPU    float64                    `json:"cpu"`
			Stress int                      `json:"stress"`
			VMs    []orchestrator.Server    `json:"vms"`
		}{
			CPU:    monitor.GetUsage(),
			Stress: stressLevel,
			VMs:    orch.GetServers(),
		})
	})

	http.HandleFunc("/cpu", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"cpu": %.2f, "stress": %d}`, monitor.GetUsage(), stressLevel)
	})

	http.HandleFunc("/stress", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			level, _ := strconv.Atoi(r.FormValue("level"))
			stressLevel = level
			applyStress(level)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{Status: "ok"})
		}
	})

	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			cfg := monitor.Config{}

			if u, err := strconv.ParseFloat(r.FormValue("upper"), 64); err == nil {
				cfg.UpperThreshold = u
			}
			if l, err := strconv.ParseFloat(r.FormValue("lower"), 64); err == nil {
				cfg.LowerThreshold = l
			}
			if i, err := strconv.Atoi(r.FormValue("interval")); err == nil {
				cfg.Interval = time.Duration(i) * time.Second
			}

			monitor.SetConfig(cfg)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cfg)
		}
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Server running on http://localhost%s", addr)
	log.Printf("HAProxy orchestration enabled")
	log.Fatal(http.ListenAndServe(addr, nil))
}

var stressProcess *exec.Cmd

func applyStress(level int) {
	if level <= 0 {
		if stressProc != nil {
			stressProc.Process.Kill()
		}
		exec.Command("pkill", "-9", "stress-ng").Run()
		log.Println("Stress stopped")
		return
	}
	log.Printf("Applying stress level %d", level)
	
	if stressProc != nil {
		stressProc.Process.Kill()
	}
	
	cmd := exec.Command("stress-ng", 
		"--cpu", fmt.Sprintf("%d", level),
		"--timeout", "3600s",
		"--backoff", "100000",
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Start()
	stressProc = cmd
	
	log.Printf("stress-ng started with PID %d", cmd.Process.Pid)
}

func scaleUp() {
	log.Println("Scaling UP - CPU above threshold")
	vmName := fmt.Sprintf("aux-%d", time.Now().Unix())
	vmIP := fmt.Sprintf("192.168.1.%d", 50+orch.GetActiveCount())
	vmPort := 8000

	if err := orch.AddServer(vmName, vmIP, vmPort); err != nil {
		log.Printf("Error adding server: %v", err)
	}

	scriptPath := "./scripts/crear_vm.sh"
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/c", scriptPath, vmName).Run()
	} else {
		log.Printf("Running: sh %s %s", scriptPath, vmName)
		exec.Command("sh", scriptPath, vmName).Run()
	}
}

func scaleDown() {
	log.Println("Scaling DOWN - CPU below threshold")
	servers := orch.GetServers()
	if len(servers) == 0 {
		return
	}

	vm := servers[len(servers)-1]
	if err := orch.RemoveServer(vm.Name); err != nil {
		log.Printf("Error removing server: %v", err)
	}

	scriptPath := "./scripts/eliminar_vm.sh"
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/c", scriptPath, vm.Name).Run()
	} else {
		exec.Command("sh", scriptPath, vm.Name).Run()
	}
}