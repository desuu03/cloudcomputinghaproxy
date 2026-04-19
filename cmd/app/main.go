package main

import (
	"App_Servidor_Imagenes/pkg/images"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

type PageData struct {
	Images      []images.ImageInfo
	Hostname    string
	CourseName  string
	StudentName string
	CPUUsage    float64
	StressLevel int
}

// Umbrales y configuración
var (
	upperThreshold = 80.0 // %
	lowerThreshold = 20.0 // %
	interval       = time.Minute
	stressLevel    = 0
)

type VMStatus struct {
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Active bool   `json:"active"`
}

var vmList []VMStatus // lista de VMs activas

func main() {
	port := 8000
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}

	// Obtener directorio del ejecutable
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("No se pudo obtener la ruta del ejecutable: %v", err)
		exePath = ""
	}
	exeDir := filepath.Dir(exePath)

	// Hostname
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("No se pudo obtener el hostname: %v", err)
		hostname = "localhost"
	}

	// Plantilla
	templatePath := filepath.Join(exeDir, "templates", "index.html")

	// Si no existe en el directorio del ejecutable, usar ruta relativa al proyecto
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		templatePath = "templates/index.html"
	}

	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"safeURL": func(u string) template.URL {
			return template.URL(u)
		},
	}).ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("Error al cargar la plantilla %s: %v", templatePath, err)
	}

	// Endpoint para estado del sistema
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		usage := getCPUUsage()
		w.Header().Set("Content-Type", "application/json")

		// Devolver CPU, nivel de estrés y lista de VMs
		data := struct {
			CPU    float64    `json:"cpu"`
			Stress int        `json:"stress"`
			VMs    []VMStatus `json:"vms"`
		}{
			CPU:    usage,
			Stress: stressLevel,
			VMs:    vmList,
		}

		json.NewEncoder(w).Encode(data)
	})

	// Endpoint principal
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		imageDir := filepath.Join(exeDir, "ImagenesApp")

		// Si no existe en el directorio del ejecutable, usar ruta relativa al proyecto
		if _, err := os.Stat(imageDir); os.IsNotExist(err) {
			imageDir = "ImagenesApp"
		}

		imageInfos, err := images.GetRandomBase64Images(imageDir, 3)
		if err != nil {
			log.Printf("Error al cargar imágenes: %v", err)
			http.Error(w, "Error al cargar imágenes: "+err.Error(), http.StatusInternalServerError)
			return
		}

		usage := getCPUUsage()

		data := PageData{
			Images:      imageInfos,
			Hostname:    hostname,
			CourseName:  "Cloud Computing",
			StudentName: "David Alfonso Posso Cano y Santiago Navarro",
			CPUUsage:    usage,
			StressLevel: stressLevel,
		}

		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Error al ejecutar la plantilla: %v", err)
		}
	})

	// Modificar el endpoint de stress
	http.HandleFunc("/stress", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			levelStr := r.FormValue("level")
			level, _ := strconv.Atoi(levelStr)
			stressLevel = level
			adjustStress(level)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"ok","stress":%d}`, level)
		}
	})

	// Endpoint para configurar umbrales e intervalos
	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			upperStr := r.FormValue("upper")
			lowerStr := r.FormValue("lower")
			intervalStr := r.FormValue("interval")

			if val, err := strconv.ParseFloat(upperStr, 64); err == nil {
				upperThreshold = val
			}
			if val, err := strconv.ParseFloat(lowerStr, 64); err == nil {
				lowerThreshold = val
			}
			if val, err := strconv.Atoi(intervalStr); err == nil {
				interval = time.Duration(val) * time.Second
			}

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"ok","upper":%.2f,"lower":%.2f,"interval":%d}`,
				upperThreshold, lowerThreshold, int(interval.Seconds()))
		}
	})

	// Nuevo endpoint para CPU en tiempo real
	http.HandleFunc("/cpu", func(w http.ResponseWriter, r *http.Request) {
		usage := getCPUUsage()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"cpu": %.2f, "stress": %d}`, usage, stressLevel)
	})

	// Lanzar monitoreo de CPU
	go monitorCPU()

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Servidor corriendo en http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Error al iniciar el servidor: %v\n", err)
		os.Exit(1)
	}

}

// ------------------- FUNCIONES AUXILIARES -------------------

func getCPUUsage() float64 {
	usage, _ := cpu.Percent(0, false)
	if len(usage) > 0 {
		return usage[0]
	}
	return 0.0
}

func monitorCPU() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var highCount, lowCount int

	for range ticker.C {
		val := getCPUUsage()
		if val > upperThreshold {
			highCount++
			lowCount = 0
			if highCount*5 >= int(interval.Seconds()) {
				scaleUp()
				highCount = 0
			}
		} else if val < lowerThreshold {
			lowCount++
			highCount = 0
			if lowCount*5 >= int(interval.Seconds()) {
				scaleDown()
				lowCount = 0
			}
		} else {
			highCount, lowCount = 0, 0
		}

	}

}

func scaleUp() {
	log.Println("Escalando hacia arriba...")
	vmName := fmt.Sprintf("vm_%d", time.Now().Unix())
	vmIP := "192.168.1.50" // aquí deberías obtener la IP real de la VM creada
	vmList = append(vmList, VMStatus{Name: vmName, IP: vmIP, Active: true})

	exec.Command("sh", "./scripts/crear_vm.sh").Run()
	exec.Command("sh", "./scripts/haproxy_add.sh", vmName, vmIP).Run()
}

func scaleDown() {
	log.Println("Escalando hacia abajo...")
	if len(vmList) > 0 {
		vm := vmList[len(vmList)-1]
		vmList = vmList[:len(vmList)-1]

		exec.Command("sh", "./scripts/haproxy_remove.sh", vm.Name).Run()
		exec.Command("sh", "./scripts/eliminar_vm.sh", vm.Name).Run()
	}
}

func adjustStress(level int) {
	log.Printf("Ajustando stress-ng al nivel %d...\n", level)
	// Ejemplo: lanzar stress-ng con X workers
	exec.Command("sh", "-c", fmt.Sprintf("stress-ng --cpu %d --timeout 60s", level)).Run()
}
