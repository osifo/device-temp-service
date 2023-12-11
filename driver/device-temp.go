// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 Canonical Ltd
// Copyright (C) 2023 Tilte Labs Ltd
//
// SPDX-License-Identifier: Apache-2.0

// Package driver provides a simple example implementation of
// ProtocolDriver interface.
package driver

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"reflect"
	"strconv"
	"time"
	"math/rand"
	// "os/exec"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
	gometrics "github.com/rcrowley/go-metrics"

	"github.com/edgexfoundry/device-sdk-go/v3/example/config"
	"github.com/edgexfoundry/device-sdk-go/v3/pkg/interfaces"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v3/pkg/models"
)

const readCommandsExecutedName = "ReadCommandsExecuted"

type TemperatureDriver struct {
	sdk                  interfaces.DeviceServiceSDK
	lc                   logger.LoggingClient
	asyncCh              chan<- *sdkModels.AsyncValues
	deviceCh             chan<- []sdkModels.DiscoveredDevice
	switchButton         bool
	xRotation            int32
	yRotation            int32
	zRotation            int32
	counter              interface{}
	stringArray          []string
	readCommandsExecuted gometrics.Counter
	serviceConfig        *config.ServiceConfig
}

func getImageBytes(imgFile string, buf *bytes.Buffer) error {
	// Read existing image from file
	img, err := os.Open(imgFile)
	if err != nil {
		return err
	}
	defer img.Close()

	// TODO: Attach MediaType property, determine if decoding
	//  early is required (to optimize edge processing)

	// Expect "png" or "jpeg" image type
	imageData, imageType, err := image.Decode(img)
	if err != nil {
		return err
	}
	// Finished with file. Reset file pointer
	_, err = img.Seek(0, 0)
	if err != nil {
		return err
	}
	if imageType == "jpeg" {
		err = jpeg.Encode(buf, imageData, nil)
		if err != nil {
			return err
		}
	} else if imageType == "png" {
		err = png.Encode(buf, imageData)
		if err != nil {
			return err
		}
	}
	return nil
}

// Initialize performs protocol-specific initialization for the device
// service.
func (s *TemperatureDriver) Initialize(sdk interfaces.DeviceServiceSDK) error {
	s.sdk = sdk
	s.lc = sdk.LoggingClient()
	s.asyncCh = sdk.AsyncValuesChannel()
	s.deviceCh = sdk.DiscoveredDeviceChannel()
	s.serviceConfig = &config.ServiceConfig{}
	s.counter = map[string]interface{}{
		"f1": "ABC",
		"f2": 123,
	}
	s.stringArray = []string{"foo", "bar"}

	if err := sdk.LoadCustomConfig(s.serviceConfig, "SimpleCustom"); err != nil {
		return fmt.Errorf("unable to load 'SimpleCustom' custom configuration: %s", err.Error())
	}

	s.lc.Infof("Custom config is: %v", s.serviceConfig.SimpleCustom)

	if err := s.serviceConfig.SimpleCustom.Validate(); err != nil {
		return fmt.Errorf("'SimpleCustom' custom configuration validation failed: %s", err.Error())
	}

	if err := sdk.ListenForCustomConfigChanges(
		&s.serviceConfig.SimpleCustom.Writable,
		"SimpleCustom/Writable", s.ProcessCustomConfigChanges); err != nil {
		return fmt.Errorf("unable to listen for changes for 'SimpleCustom.Writable' custom configuration: %s", err.Error())
	}

	s.readCommandsExecuted = gometrics.NewCounter()

	var err error
	metricsManger := sdk.MetricsManager()
	if metricsManger != nil {
		err = metricsManger.Register(readCommandsExecutedName, s.readCommandsExecuted, nil)
	} else {
		err = errors.New("metrics manager not available")
	}

	if err != nil {
		return fmt.Errorf("unable to register metric %s: %s", readCommandsExecutedName, err.Error())
	}

	s.lc.Infof("Registered %s metric for collection when enabled", readCommandsExecutedName)

	return nil
}

// ProcessCustomConfigChanges ...
func (s *TemperatureDriver) ProcessCustomConfigChanges(rawWritableConfig interface{}) {
	updated, ok := rawWritableConfig.(*config.SimpleWritable)
	if !ok {
		s.lc.Error("unable to process custom config updates: Can not cast raw config to type 'SimpleWritable'")
		return
	}

	s.lc.Info("Received configuration updates for 'SimpleCustom.Writable' section")

	previous := s.serviceConfig.SimpleCustom.Writable
	s.serviceConfig.SimpleCustom.Writable = *updated

	if reflect.DeepEqual(previous, *updated) {
		s.lc.Info("No changes detected")
		return
	}

	// Now check to determine what changed.
	// In this example we only have the one writable setting,
	// so the check is not really need but left here as an example.
	// Since this setting is pulled from configuration each time it is need, no extra processing is required.
	// This may not be true for all settings, such as external host connection info, which
	// may require re-establishing the connection to the external host for example.
	if previous.DiscoverSleepDurationSecs != updated.DiscoverSleepDurationSecs {
		s.lc.Infof("DiscoverSleepDurationSecs changed to: %d", updated.DiscoverSleepDurationSecs)
	}
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (s *TemperatureDriver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest) (res []*sdkModels.CommandValue, err error) {
	s.lc.Debugf("TemperatureDriver.HandleReadCommands: protocols: %v resource: %v attributes: %v", protocols, reqs[0].DeviceResourceName, reqs[0].Attributes)

	res = make([]*sdkModels.CommandValue, 0)
	for _, req := range reqs {
		var cv *sdkModels.CommandValue
		switch req.DeviceResourceName {
			case "Temperature":
				// temp_out, err := exec.Command("cat", "sys/class/thermal/thermal_zone11/temp").Output()
				// if err != nil {
				// 	s.lc.Debugf(" ========== an error has occurred ========== ", err)
				// 	return nil, err
				// }

				// s.lc.Debugf(" ========== output ========== ", temp_out)

				// temp_val, err := strconv.ParseInt(string(temp_out[:len(temp_out)-1]), 10, 64)

				// if err == nil {
				// 	cv, _ = sdkModels.NewCommandValue(req.DeviceResourceName, common.ValueTypeInt64, int64(temp_val))
				// 	res[0] = cv
				// } else {
				// 	return nil, err
				// }
				temp_val := rand.Int63n(70)

				cv, _ = sdkModels.NewCommandValue(req.DeviceResourceName, common.ValueTypeInt64, int64(temp_val))
				s.lc.Debugf(" ========== output ========== ", cv)

			case "Motion":
				err := fmt.Sprintf("[DEBUG] ======== The CommandValue received (%v) does not have a handler logic.", req.DeviceResourceName)
				fmt.Printf(err)

				return nil, errors.New(err)
		}

		res = append(res, cv)
	}

	s.readCommandsExecuted.Inc(1)

	return
}

// HandleWriteCommands passes a slice of CommandRequest struct each representing
// a ResourceOperation for a specific device resource.
// Since the commands are actuation commands, params provide parameters for the individual
// command.
func (s *TemperatureDriver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest,
	params []*sdkModels.CommandValue) error {
	var err error

	for i, r := range reqs {
		s.lc.Debugf("TemperatureDriver.HandleWriteCommands: protocols: %v, resource: %v, parameters: %v, attributes: %v", protocols, reqs[i].DeviceResourceName, params[i], reqs[i].Attributes)
		switch r.DeviceResourceName {
		case "SwitchButton":
			if s.switchButton, err = params[i].BoolValue(); err != nil {
				err := fmt.Errorf("TemperatureDriver.HandleWriteCommands; the data type of parameter should be Boolean, parameter: %s", params[0].String())
				return err
			}
		case "Xrotation":
			if s.xRotation, err = params[i].Int32Value(); err != nil {
				err := fmt.Errorf("TemperatureDriver.HandleWriteCommands; the data type of parameter should be Int32, parameter: %s", params[i].String())
				return err
			}
		case "Yrotation":
			if s.yRotation, err = params[i].Int32Value(); err != nil {
				err := fmt.Errorf("TemperatureDriver.HandleWriteCommands; the data type of parameter should be Int32, parameter: %s", params[i].String())
				return err
			}
		case "Zrotation":
			if s.zRotation, err = params[i].Int32Value(); err != nil {
				err := fmt.Errorf("TemperatureDriver.HandleWriteCommands; the data type of parameter should be Int32, parameter: %s", params[i].String())
				return err
			}
		case "StringArray":
			if s.stringArray, err = params[i].StringArrayValue(); err != nil {
				err := fmt.Errorf("TemperatureDriver.HandleWriteCommands; the data type of parameter should be string array, parameter: %s", params[i].String())
				return err
			}
		case "Uint8Array":
			v, err := params[i].Uint8ArrayValue()
			if err == nil {
				s.lc.Debugf("Uint8 array value from write command: ", v)
			} else {
				return err
			}
		case "Counter":
			if s.counter, err = params[i].ObjectValue(); err != nil {
				err := fmt.Errorf("TemperatureDriver.HandleWriteCommands; the data type of parameter should be Object, parameter: %s", params[i].String())
				return err
			}
		}
	}

	return nil
}

// Stop the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if supported).
func (s *TemperatureDriver) Stop(force bool) error {
	// Then Logging Client might not be initialized
	if s.lc != nil {
		s.lc.Debugf("TemperatureDriver.Stop called: force=%v", force)
	}
	return nil
}

func (s *TemperatureDriver) Start() error {
	return nil
}

// AddDevice is a callback function that is invoked
// when a new Device associated with this Device Service is added
func (s *TemperatureDriver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	s.lc.Debugf("a new Device is added: %s", deviceName)
	return nil
}

// UpdateDevice is a callback function that is invoked
// when a Device associated with this Device Service is updated
func (s *TemperatureDriver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	s.lc.Debugf("Device %s is updated", deviceName)
	return nil
}

// RemoveDevice is a callback function that is invoked
// when a Device associated with this Device Service is removed
func (s *TemperatureDriver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	s.lc.Debugf("Device %s is removed", deviceName)
	return nil
}

// Discover triggers protocol specific device discovery, which is an asynchronous operation.
// Devices found as part of this discovery operation are written to the channel devices.
func (s *TemperatureDriver) Discover() error {
	proto := make(map[string]models.ProtocolProperties)
	proto["other"] = map[string]any{"Address": "simple02", "Port": 301}

	device2 := sdkModels.DiscoveredDevice{
		Name:        "Simple-Device02",
		Protocols:   proto,
		Description: "found by discovery",
		Labels:      []string{"auto-discovery"},
	}

	proto = make(map[string]models.ProtocolProperties)
	proto["other"] = map[string]any{"Address": "simple03", "Port": 399}

	device3 := sdkModels.DiscoveredDevice{
		Name:        "Simple-Device03",
		Protocols:   proto,
		Description: "found by discovery",
		Labels:      []string{"auto-discovery"},
	}

	res := []sdkModels.DiscoveredDevice{device2, device3}

	time.Sleep(time.Duration(s.serviceConfig.SimpleCustom.Writable.DiscoverSleepDurationSecs) * time.Second)
	s.deviceCh <- res
	return nil
}

func (s *TemperatureDriver) ValidateDevice(device models.Device) error {
	protocol, ok := device.Protocols["other"]
	if !ok {
		return errors.New("missing 'other' protocols")
	}

	addr, ok := protocol["Address"]
	if !ok {
		return errors.New("missing 'Address' information")
	} else if addr == "" {
		return errors.New("address must not empty")
	}

	port, ok := protocol["Port"]
	if !ok {
		return errors.New("missing 'Port' information")
	} else {
		portString := fmt.Sprintf("%v", port)
		_, err := strconv.ParseUint(portString, 10, 64)
		if err != nil {
			return errors.New("port must be a number")
		}
	}

	return nil
}
