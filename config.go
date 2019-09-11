// go-config - Library for reading cacophony config files.
// Copyright (C) 2018, The Cacophony Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package config

import (
	"context"
	"errors"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type Config struct {
	v                *viper.Viper
	fileLock         *flock.Flock
	accessedSections []string //TODO record each section accessed for restarting service purpose
}

type Device struct {
	Group  string
	ID     int
	Name   string
	Server string
}

type Location struct {
	Accuracy  int
	Altitude  int
	Latitude  float32
	Longitude float32
}

var DefaultLocation = Location{
	Latitude:  -43.5321,
	Longitude: 172.6362,
}

type Windows struct {
	StartRecording string `mapstructure:"start-recording"`
	StopRecording  string `mapstructure:"stop-recording"`
	PowerOn        string `mapstructure:"power-on"`
	PowerOff       string `mapstructure:"power-off"`
}

var DefaultWindows = Windows{
	StartRecording: "-30m",
	StopRecording:  "+30m",
	PowerOn:        "12:00",
	PowerOff:       "12:00",
}

type TestHosts struct {
	URLs         []string
	PingWaitTime time.Duration `mapstructure:"ping-wait-time"`
	PingRetries  int           `mapstructure:"ping-retries"`
}

var DefaultTestHosts = TestHosts{
	URLs:         []string{"1.1.1.1", "8.8.8.8"},
	PingWaitTime: time.Second * 30,
	PingRetries:  5,
}

type Battery struct {
	EnableVoltageReadings bool   `mapstructure:"enable-voltage-readings"`
	NoBattery             uint16 `mapstructure:"no-battery-reading"`
	LowBattery            uint16 `mapstructure:"low-battery-reading"`
	FullBattery           uint16 `mapstructure:"full-battery-reading"`
}

type Modemd struct {
	TestInterval      time.Duration `mapstructure:"test-interval"`
	InitialOnDuration time.Duration `mapstructure:"initial-on-duration"`
	FindModemTimeout  time.Duration `mapstructure:"find-modem-timeout"`
	ConnectionTimeout time.Duration `mapstructure:"connection-timeout"`
	RequestOnDuration time.Duration `mapstructure:"request-on-duration"`
	Modems            []modem       `mapstructure:"modems"`
}

type modem struct {
	Name            string `mapstructure:"name"`
	NetDev          string `mapstructure:"net-dev"`
	VendorProductID string `mapstructure:"vendor-product-id"`
}

var DefaultModemd = Modemd{
	TestInterval:      time.Minute * 5,
	InitialOnDuration: time.Hour * 24,
	FindModemTimeout:  time.Minute * 2,
	ConnectionTimeout: time.Minute,
	RequestOnDuration: time.Hour * 24,
	Modems: []modem{
		modem{Name: "Huawei 4G modem", NetDev: "eth1", VendorProductID: "12d1:14db"},
		modem{Name: "Spark 3G modem", NetDev: "usb0", VendorProductID: "19d2:1405"},
	},
}

type Lepton struct {
	SPISpeed    int64  `mapstructure:"spi-speed"`
	FrameOutput string `mapstructure:"frame-output"`
}

var DefaultLepton = Lepton{
	SPISpeed:    2000000,
	FrameOutput: "/var/run/lepton-frames",
}

type ThermalRecorder struct {
	OutputDir      string `mapstructure:"output-dir"`
	MinDiskSpaceMB uint64 `mapstructure:"min-disk-space-mb"`
	MinSecs        int    `mapstructure:"min-secs"`
	MaxSecs        int    `mapstructure:"max-secs"`
	PreviewSecs    int    `mapstructure:"preview-secs"`
}

var DefaultThermalRecorder = ThermalRecorder{
	MaxSecs:        600,
	MinSecs:        10,
	PreviewSecs:    3,
	MinDiskSpaceMB: 200,
	OutputDir:      "/var/spool/cptv",
}

type ThermalMotion struct {
	DynamicThreshold bool   `mapstructure:"min-secs"`
	TempThresh       uint16 `mapstructure:"temp-thresh"`
	DeltaThresh      uint16 `mapstructure:"delta-thresh"`
	CountThresh      int    `mapstructure:"count-thresh"`
	FrameCompareGap  int    `mapstructure:"frame-compare-gap"`
	UseOneDiffOnly   bool   `mapstructure:"use-one-diff-only"`
	TriggerFrames    int    `mapstructure:"trigger-frames"`
	WarmerOnly       bool   `mapstructure:"warmer-only"`
	EdgePixels       int    `mapstructure:"edge-pixels"`
	Verbose          bool   `mapstructure:"verbose"`
}

var DefaultThermalMotion = ThermalMotion{
	DynamicThreshold: true,
	TempThresh:       2900,
	DeltaThresh:      50,
	CountThresh:      3,
	FrameCompareGap:  45,
	Verbose:          false,
	TriggerFrames:    2,
	UseOneDiffOnly:   true,
	WarmerOnly:       true,
	EdgePixels:       1,
}

type ThermalThrottler struct {
	Activate   bool
	BucketSize time.Duration `mapstructure:"bucket-size"`
	MinRefill  time.Duration `mapstructure:"min-refill"`
}

var DefaultThermalThrottler = ThermalThrottler{
	Activate:   true,
	BucketSize: 10 * time.Minute,
	MinRefill:  10 * time.Minute,
}

type Ports struct {
	Managementd int
}

var DefaultPorts = Ports{
	Managementd: 80,
}

type Secrets struct {
	DevicePassword string `mapstructure:"device-password"`
}

type GPIO struct {
	ThermalCameraPower string `mapstructure:"thermal-camera-power"`
	ModemPower         string `mapstructure:"modem-power"`
}

var DefaultGPIO = GPIO{
	ThermalCameraPower: "GPIO23",
	ModemPower:         "GPIO22",
}

const (
	DeviceKey           = "device"
	WindowsKey          = "windows"
	LocationKey         = "location"
	TestHostsKey        = "test-hosts"
	BatteryKey          = "battery"
	ModemdKey           = "modemd"
	LeptonKey           = "lepton"
	ThermalRecorderKey  = "thermal-recorder"
	ThermalMotionKey    = "thermal-motion"
	ThermalThrottlerKey = "thermal-throttler"
	PortsKey            = "ports"
	SecretsKey          = "secrets"
	GPIOKey             = "gpio"

	DefaultConfigDir = "/etc/cacophony"
	configFileName   = "config.toml"
	lockRetryDelay   = 678 * time.Millisecond
)

// Helpers for testign purposes
var fs = afero.NewOsFs()
var now = time.Now
var lockFilePath = func(configFile string) string {
	return configFile + ".lock"
}
var lockTimeout = 10 * time.Second

// New created a new config and loads files from the given directory
func New(dir string) (*Config, error) {
	// TODO Take service name and restart service if config changes
	configFile := path.Join(dir, configFileName)
	c := &Config{
		v:        viper.New(),
		fileLock: flock.New(lockFilePath(configFile)),
	}
	c.v.SetFs(fs)
	c.v.SetConfigFile(configFile)
	if err := c.getFileLock(); err != nil {
		return nil, err
	}
	defer c.fileLock.Unlock()
	if err := c.v.ReadInConfig(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) Unmarshal(key string, raw interface{}) error {
	return c.v.UnmarshalKey(key, raw)
}

func (c *Config) Set(key string, value interface{}) error {
	if err := c.getFileLock(); err != nil {
		return err
	}
	defer c.fileLock.Unlock()
	if err := c.Update(); err != nil {
		return err
	}
	kind := reflect.ValueOf(value).Kind()
	if kind == reflect.Struct || kind == reflect.Ptr {
		return c.setStruct(key, value)
	}
	c.set(key, value)
	return c.writeConfig()
}

func (c *Config) Update() error {
	if err := c.getFileLock(); err != nil {
		return err
	}
	defer c.fileLock.Unlock()
	return c.v.ReadInConfig()
}

// TODO Only update if given time is after the "udpate" field of the section updating and set "update" field to given time if updating
func (c *Config) StrictSet(key string, value interface{}, time time.Time) error {
	return nil
}

var errNoFileLock = errors.New("failed to get lock on file")

func (c *Config) getFileLock() error {
	lockCtx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()
	locked, err := c.fileLock.TryLockContext(lockCtx, lockRetryDelay)
	if err != nil {
		return err
	} else if !locked {
		return errNoFileLock
	}
	return nil
}

func interfaceToMap(value interface{}) (m map[string]interface{}, err error) {
	err = mapstructure.Decode(value, &m)
	return
}

func (c *Config) setStruct(key string, value interface{}) error {
	m, err := interfaceToMap(value)
	if err != nil {
		return err
	}
	c.set(key, m)
	return c.writeConfig()
}

func (c *Config) set(key string, value interface{}) {
	c.v.Set(key, value)
	c.v.Set(strings.Split(key, ".")[0]+".updated", now())
}

func (c *Config) writeConfig() error {
	return c.v.WriteConfig()
}

func (c *Config) Get(key string) interface{} {
	return c.v.Get(key)
}
