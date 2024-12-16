package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fiddeb/elcentral/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.bug.st/serial"
	"go.uber.org/zap"
)

const version = "0.0.1"

func main() {
	// Definiera flaggor
	showVersion := flag.Bool("version", false, "Show version and exit")
	serialPortPath := flag.String("serialPort", "/dev/tty.usbserial-A19JSWCW", "Path to the serial port")
	debug := flag.Bool("debug", false, "Enable debug logging")
	

	// Parsning av flaggor
	flag.Parse()

	// Visa versionsnummer och avsluta om flaggan är satt
	if *showVersion {
		fmt.Println("Version:", version)
		return
	}

	// Använd flaggorna
	if *debug {
		fmt.Println("Debugging enabled")
		// Aktivera debug-läge i loggningssystemet, t.ex. logger.SetLevel(zap.DebugLevel)
	}

	fmt.Println("Serial port:", *serialPortPath)

	var logger *zap.Logger
	var err error

	if *debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	// Hantera avslutningssignaler
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Seriell portkonfiguration
	serialConfig := &serial.Mode{
		BaudRate: 115200, // Kontrollera att detta är rätt baudrate
	}

	// Anslut till serial port med konfiguration
	serialPort, err := serial.Open(*serialPortPath, serialConfig)
	if err != nil {
		sugar.Fatalf("Error opening serial port: %v", err)
	}
	defer serialPort.Close()

	// Starta Prometheus HTTP-server
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
	go func() {
		sugar.Infof("Starting Prometheus HTTP server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			sugar.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Läs data från serieporten
	go metrics.ReadSerialData(serialPort, sugar)

	<-quit
	sugar.Info("Shutting down application...")
}