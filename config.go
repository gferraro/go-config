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
	v        *viper.Viper
	fileLock *flock.Flock
}

type Device struct {
	Group    string
	ID       int
	Name     string
	Password string
	Server   string
}

type Location struct {
	Accuracy  int
	Altitude  int
	Latitude  float32
	Longitude float32
}

type Windows struct {
	StartRecording string `mapstructure:"start-recording"`
	StopRecording  string `mapstructure:"stop-recording"`
	PowerOn        string `mapstructure:"power-on"`
	PowerOff       string `mapstructure:"power-off"`
}

type TestHosts struct {
	URLs         []string
	PingWaitTime time.Duration `mapstructure:"ping-wait-time"`
	PingRetries  int           `mapstructure:"ping-retries"`
}

const (
	DeviceKey    = "device"
	WindowsKey   = "windows"
	LocationKey  = "location"
	TestHostsKey = "test-hosts"

	DefaultConfigDir = "/etc/cacophony"
	configFileName   = "config.toml"
	lockRetryDelay   = 678 * time.Millisecond
)

var defaultWindows = Windows{
	StartRecording: "-30m",
	StopRecording:  "+30m",
	PowerOn:        "-45m",
	PowerOff:       "+45m",
}
var defaultLocation = Location{
	Latitude:  12,
	Longitude: 12,
}
var defaultTestHosts = TestHosts{
	URLs:         []string{"1.1.1.1", "8.8.8.8"},
	PingRetries:  5,
	PingWaitTime: time.Second * 30,
}
var defaultSettings = map[string]interface{}{
	WindowsKey:   defaultWindows,
	LocationKey:  defaultLocation,
	TestHostsKey: defaultTestHosts,
}

// Helpers for testign purposes
var fs = afero.NewOsFs()
var now = time.Now
var lockFilePath = func(configFile string) string {
	return configFile + ".lock"
}
var lockTimeout = 10 * time.Second

// New created a new config and loads files from the given directory
func New(dir string) (*Config, error) {
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
	if err := c.setDefaults(); err != nil {
		return nil, err
	}
	if err := c.v.ReadInConfig(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) GetDevice() (d *Device, err error) {
	err = c.Unmarshal(DeviceKey, &d)
	return
}

func (c *Config) GetWindows() (w *Windows, err error) {
	err = c.Unmarshal(WindowsKey, &w)
	return
}

func (c *Config) GetLocation() (l *Location, err error) {
	err = c.Unmarshal(LocationKey, &l)
	return
}

func (c *Config) GetTestHosts() (u *TestHosts, err error) {
	err = c.Unmarshal(TestHostsKey, &u)
	return
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

func (c *Config) setDefaults() error {
	for k, v := range defaultSettings {
		m, err := interfaceToMap(v)
		if err != nil {
			return err
		}
		c.v.SetDefault(k, m)
	}
	return nil
}

func (c *Config) writeConfig() error {
	return c.v.WriteConfig()
}

func (c *Config) Get(key string) interface{} {
	return c.v.Get(key)
}

func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

func (c *Config) GetFloat64(key string) float64 {
	return c.v.GetFloat64(key)
}

func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

func (c *Config) GetTime(key string) time.Time {
	return c.v.GetTime(key)
}

func (c *Config) GetDuration(key string) time.Duration {
	return c.v.GetDuration(key)
}

func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}
