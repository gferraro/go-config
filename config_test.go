package config

import (
	"io/ioutil"
	"log"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/TheCacophonyProject/window"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wawandco/fako"
)

const tomlFileString = `
`
const (
	testTomlName        = "test.toml"
	testTomlDefaultName = "test-default.toml"
	testTomlFileDir     = "/"
)

type Voltages struct {
	Enable      bool   // Enable reading voltage through ATtiny
	NoBattery   uint16 // If voltage reading is less than this it is not powered by a battery
	LowBattery  uint16 // Voltage of a low battery
	FullBattery uint16 // Voltage of a full battery
}

type AttinyConfig struct {
	OnWindow *window.Window
	Voltages Voltages
}

func newFs(t *testing.T, file string) {
	fs = afero.NewMemMapFs()
	b, err := ioutil.ReadFile(file)
	require.NoError(t, err)
	filePath := path.Join(testTomlFileDir, configFileName)
	require.NoError(t, afero.WriteFile(fs, filePath, b, 0644))
}

func printConfigFile(dir string) {
	filePath := path.Join(dir, configFileName)
	b, err := afero.ReadFile(fs, filePath)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(b))
}

func TestReadingConfigInDir(t *testing.T) {
	newFs(t, testTomlName)
	conf, err := New(testTomlFileDir)
	require.NoError(t, err)

	// Get device struct
	d := &Device{}
	require.NoError(t, conf.Unmarshal(DeviceKey, d))

	rawConfig, err := New(testTomlFileDir)
	assert.NoError(t, err)

	voltagesRaw := &Voltages{}
	rawConfig.Unmarshal("attiny.voltages", voltagesRaw)
	windowRaw := &Windows{}
	rawConfig.Unmarshal(WindowsKey, windowRaw)
	locationRaw := &Location{}
	rawConfig.Unmarshal(LocationKey, locationRaw)

	_, err = window.New(
		windowRaw.PowerOn,
		windowRaw.PowerOff,
		float64(locationRaw.Latitude),
		float64(locationRaw.Longitude))
	require.NoError(t, err)

}

func TestWriting(t *testing.T) {
	newFs(t, testTomlName)
	conf, err := New(testTomlFileDir)
	require.NoError(t, err)

	d := randomDevice()
	w := randomWindows()
	l := randomLocation()
	u := randomURLs()
	require.NoError(t, conf.Set(DeviceKey, d))
	require.NoError(t, conf.Set(WindowsKey, w))
	require.NoError(t, conf.Set(LocationKey, &l))
	require.NoError(t, conf.Set(URLsKey, &u))

	d.Name = randString(10)
	conf.Set(DeviceKey+".name", d.Name)

	conf, err = New(testTomlFileDir)
	require.NoError(t, err)
	d2, err := conf.GetDevice()
	require.NoError(t, err)
	w2, err := conf.GetWindows()
	require.NoError(t, err)
	l2, err := conf.GetLocation()
	require.NoError(t, err)
	u2, err := conf.GetURLs()
	require.NoError(t, err)

	require.Equal(t, d, *d2)
	require.Equal(t, w, *w2)
	require.Equal(t, l, *l2)
	require.Equal(t, u, *u2)
	require.Equal(t, d.Name, conf.Get(DeviceKey+".name"))
}

func TestDefault(t *testing.T) {
	newFs(t, testTomlDefaultName)
	conf, err := New(testTomlFileDir)
	require.NoError(t, err)

	w := Windows{}
	l := Location{}
	u := URLs{}
	require.NoError(t, conf.Unmarshal(WindowsKey, &w))
	require.NoError(t, conf.Unmarshal(LocationKey, &l))
	require.NoError(t, conf.Unmarshal(URLsKey, &u))

	require.Equal(t, w, defaultWindows)
	require.Equal(t, l, defaultLocation)
	require.Equal(t, u, defaultURLs)
}

func TestSettingUpdated(t *testing.T) {
	newNow()
	newFs(t, testTomlName)
	conf, err := New(testTomlFileDir)
	require.NoError(t, err)

	require.Equal(t, conf.Get(DeviceKey+".updated"), int64(0))
	require.NoError(t, conf.Set(DeviceKey, randomDevice()))
	require.Equal(t, conf.Get(DeviceKey+".updated"), now())
	newNow()
	require.NoError(t, conf.Set(DeviceKey, randomDevice()))
	require.Equal(t, conf.Get(DeviceKey+".updated"), now())
}

func newNow() {
	n := time.Now()
	now = func() time.Time {
		return n
	}
}

func randomDevice() (d Device) {
	fako.Fuzz(&d)
	return
}

func randomWindows() (w Windows) {
	fako.Fuzz(&w)
	return
}

func randomLocation() (l Location) {
	fako.Fuzz(&l)
	return
}

func randomURLs() URLs {
	return URLs{
		PingTests: []string{randString(10), randString(20), randString(15)},
		ProdAPI:   randString(13),
		TestAPI:   randString(14),
	}
}

// Random string
const (
	chars       = "abcdefghijklmnopqrstuvwxyz0123456789"
	charIdxBits = 6                  // 6 bits to represent a char index
	charIdxMax  = 63 / charIdxBits   // # of char indices fitting in 63 bits
	charIdxMask = 1<<charIdxBits - 1 // All 1-bits, as many as charIdxBits
)

var randSrc = rand.NewSource(time.Now().UnixNano())

func randString(n int) string {
	b := make([]byte, n)
	// A randSrc.Int63() generates 63 random bits, enough for charIdxMax characters!
	for i, cache, remain := n-1, randSrc.Int63(), charIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc.Int63(), charIdxMax
		}
		if idx := int(cache & charIdxMask); idx < len(chars) {
			b[i] = chars[idx]
			i--
		}
		cache >>= charIdxBits
		remain--
	}
	return string(b)
}
