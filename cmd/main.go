// -*- Mode: Go; indent-tabs-mode: t -*-

package main

import (
	"github.com/osifo/device-temp-service"
	"github.com/osifo/device-temp-service/driver"
	"github.com/edgexfoundry/device-sdk-go/v3/pkg/startup"
)

const (
	serviceName string = "device-temp-service"
)

func main() {
	sd := driver.TemperatureDriver{}
	startup.Bootstrap(serviceName, device.Version, &sd)
}
