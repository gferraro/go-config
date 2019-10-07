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
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	toml "github.com/pelletier/go-toml"
)

type Config struct {
	v                *viper.Viper
	fileLock         *flock.Flock
	accessedSections map[string]struct{} //TODO record each section accessed for restarting service purpose
}

const (
	DefaultConfigDir = "/etc/cacophony"
	ConfigFileName   = "config.toml"
	lockRetryDelay   = 678 * time.Millisecond
	TimeFormat       = time.RFC3339
)

type section struct {
	key         string
	mapToStruct func(map[string]interface{}) (interface{}, error)
	validate    func(interface{}) error
}

type decodeHookFunc func(reflect.Type, reflect.Type, interface{}) (interface{}, error)

var allSections = map[string]section{} // each different section file has an init function that will add to this.
var allSectionDecodeHookFuncs = []mapstructure.DecodeHookFunc{}

// Helpers for testing purposes
var fs = afero.NewOsFs()
var now = time.Now
var lockFilePath = func(configFile string) string {
	return configFile + ".lock"
}
var lockTimeout = 10 * time.Second
var mapStrInterfaceType = reflect.TypeOf(map[string]interface{}{})

// New created a new config and loads files from the given directory
func New(dir string) (*Config, error) {
	// TODO Take service name and restart service if config changes
	configFile := path.Join(dir, ConfigFileName)
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

// Set can only update one section at a time.
func (c *Config) Set(key string, value interface{}) error {
	if !checkIfSectionKey(key) {
		return notSectionKeyError(key)
	}
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
	return c.v.WriteConfig()
}

// SetFromMap can only update one section at a time.
func (c *Config) SetFromMap(sectionKey string, newConfig map[string]interface{}) error {
	if !checkIfSectionKey(sectionKey) {
		return notSectionKeyError(sectionKey)
	}
	if err := c.getFileLock(); err != nil {
		return err
	}
	defer c.fileLock.Unlock()
	if err := c.Update(); err != nil {
		return err
	}

	section, ok := allSections[sectionKey]
	if !ok {
		return fmt.Errorf("no section found called '%s'", sectionKey)
	}
	newStruct, err := section.mapToStruct(newConfig)
	if err != nil {
		return err
	}

	return c.Set(sectionKey, newStruct)
}

func (c *Config) Update() error {
	if err := c.getFileLock(); err != nil {
		return err
	}
	defer c.fileLock.Unlock()
	return c.v.ReadInConfig()
}

// TODO Only update if given time is after the "udpate" field of the section updating and set "update" field to given time if updating
/*
func (c *Config) StrictSet(key string, value interface{}, time time.Time) error {
	return nil
}
*/

func (c *Config) Unset(key string) error {
	configMap := c.v.AllSettings()
	delete(configMap, key)
	tomlTree, err := toml.TreeFromMap(configMap)
	if err != nil {
		return err
	}
	configFile := c.v.ConfigFileUsed()
	// Need a new viper instance to clear old settings
	c.v = viper.New()
	c.v.SetFs(fs)
	c.v.SetConfigFile(configFile)
	var buf bytes.Buffer
	_, err = tomlTree.WriteTo(&buf)
	if err != nil {
		return err
	}
	if err := c.v.ReadConfig(bytes.NewReader(buf.Bytes())); err != nil {
		return err
	}
	c.v.Set(key+".updated", now())
	return c.v.WriteConfig()
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
	decodeHookFuncs := mapstructure.ComposeDecodeHookFunc(allSectionDecodeHookFuncs...)
	decoderConfig := mapstructure.DecoderConfig{
		DecodeHook: decodeHookFuncs,
		Result:     &m,
	}
	decoder, err := mapstructure.NewDecoder(&decoderConfig)
	if err != nil {
		return nil, err
	}
	err = decoder.Decode(value)
	return
}

func (c *Config) setStruct(key string, value interface{}) error {
	m, err := interfaceToMap(value)
	if err != nil {
		return err
	}
	c.set(key, m)
	return c.v.WriteConfig()
}

func notSectionKeyError(key string) error {
	return fmt.Errorf("'%s' is no a key for a section", key)
}

func checkIfSectionKey(key string) bool {
	_, ok := allSections[key]
	return ok
}

func (c *Config) set(key string, value interface{}) {
	c.v.Set(key, value)
	c.v.Set(strings.Split(key, ".")[0]+".updated", now())
}

func (c *Config) Get(key string) interface{} {
	return c.v.Get(key)
}

func SetFs(f afero.Fs) {
	fs = f
}

func SetLockFilePath(f func(string) string) {
	lockFilePath = f
}

func decodeStructFromMap(s interface{}, m map[string]interface{}, decodeHook decodeHookFunc) error {
	decoderConfig := mapstructure.DecoderConfig{
		Result:           s,
		WeaklyTypedInput: true,
	}
	if decodeHook != nil {
		decoderConfig.DecodeHook = decodeHook
	}
	decoder, err := mapstructure.NewDecoder(&decoderConfig)
	if err != nil {
		return err
	}
	return decoder.Decode(m)
}

func stringToTime(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(time.Time{}) {
		return data, nil
	}
	switch f {
	case reflect.TypeOf(""):
		return time.Parse(TimeFormat, data.(string))
	}
	return data, nil
}
