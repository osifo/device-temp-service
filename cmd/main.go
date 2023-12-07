// -*- Mode: Go; indent-tabs-mode: t -*-

package main

import (
	"github.com/osifo/device-temp-service"
	"github.com/osifo/device-temp-service/temperature-driver"
	"github.com/edgexfoundry/device-sdk-go/v3/pkg/startup"
)

const (
	serviceName string = "device-temp-service"
)

func main() {
	sd := driver.SimpleDriver{}
	startup.Bootstrap(serviceName, device.Version, &sd)
}
