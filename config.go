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
	"path"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type Config struct {
	v *viper.Viper
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
	StartRecording string
	StopRecording  string
	PowerOn        string
	PowerOff       string
}

type URLs struct {
	PingTests []string
	ProdAPI   string
	TestAPI   string
}

const (
	DeviceKey   = "device"
	WindowsKey  = "windows"
	LocationKey = "location"
	URLsKey     = "urls"

	configFileName = "config.toml"
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
var defaultURLs = URLs{
	PingTests: []string{"1.1.1.1", "8.8.8.8"},
	ProdAPI:   "https://api.cacophony.org.nz",
	TestAPI:   "https://api-test.cacophony.org.nz",
}
var defaultSettings = map[string]interface{}{
	WindowsKey:  defaultWindows,
	LocationKey: defaultLocation,
	URLsKey:     defaultURLs,
}

var fs = afero.NewOsFs()

// New created a new config and loads files from the given directory
func New(dir string) (*Config, error) {
	conf := &Config{v: viper.New()}
	conf.v.SetFs(fs)
	conf.v.SetConfigFile(path.Join(dir, configFileName))
	if err := conf.setDefaults(); err != nil {
		return nil, err
	}
	if err := conf.v.ReadInConfig(); err != nil {
		return nil, err
	}
	return conf, nil
}

func (c *Config) Unmarshal(key string, raw interface{}) error {
	return c.v.UnmarshalKey(key, raw)
}

func (c *Config) Set(key string, value interface{}) error {
	kind := reflect.ValueOf(value).Kind()
	if kind == reflect.Struct || kind == reflect.Ptr {
		return c.setStruct(key, value)
	}
	c.v.Set(key, value)
	return c.writeConfig()
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
	c.v.Set(key, m)
	return c.writeConfig()
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
