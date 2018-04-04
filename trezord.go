package main

import (
	"flag"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/server"
	"github.com/trezor/trezord-go/usb"
	"gopkg.in/natefinch/lumberjack.v2"
)

type udpPorts []int

func (i *udpPorts) String() string {
	res := ""
	for i, p := range *i {
		if i > 0 {
			res = res + ","
		}
		res = res + strconv.Itoa(p)
	}
	return res
}

func (i *udpPorts) Set(value string) error {
	p, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*i = append(*i, p)
	return nil
}

func main() {
	var logfile string
	var ports udpPorts

	flag.StringVar(&logfile, "l", "", "Log into a file, rotating after 5MB")
	flag.Var(&ports, "e", "Use UDP port for emulator. Can be repeated for more ports. Example: trezord-go -e 21324 -e 21326")
	flag.Parse()

	var lw io.Writer
	if logfile != "" {
		lw = &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    5, // megabytes
			MaxBackups: 3,
		}
	} else {
		lw = os.Stderr
	}

	m := memorywriter.New(2000)

	detailedLogWriter := memorywriter.New(3000)
	logWriter := io.MultiWriter(lw, m, detailedLogWriter)

	logger := log.New(logWriter, "", log.LstdFlags)
	detailedLogger := log.New(detailedLogWriter, "details: ", log.LstdFlags)

	logger.Println("trezord is starting.")

	detailedLogger.Println("Initing webusb")
	w, err := usb.InitWebUSB(logger, detailedLogger)

	if err != nil {
		logger.Fatalf("webusb: %s", err)
	}

	detailedLogger.Println("Initing hidapi")
	h, err := usb.InitHIDAPI(logger, detailedLogger)
	if err != nil {
		logger.Fatalf("hidapi: %s", err)
	}

	var b *usb.USB

	detailedLogger.Printf("UDP port count - %d\n", len(ports))
	if len(ports) > 0 {
		e, errUDP := usb.InitUDP(ports)
		if errUDP != nil {
			logger.Fatalf("emulator: %s", errUDP)
		}
		b = usb.Init(w, h, e)
	} else {
		b = usb.Init(w, h)
	}

	detailedLogger.Println("Creating HTTP server")
	s, err := server.New(b, logWriter, m, detailedLogWriter, logger, detailedLogger)
	if err != nil {
		logger.Fatalf("https: %s", err)
	}

	detailedLogger.Println("Running HTTP server")
	err = s.Run()
	if err != nil {
		logger.Fatalf("https: %s", err)
	}

	detailedLogger.Println("Main ended successfully")
}
