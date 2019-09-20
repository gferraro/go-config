package config

import (
	"context"
	"log"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wawandco/fako"
)

const (
	testTomlName        = "test.toml"
	testTomlDefaultName = "test-default.toml"
	testTomlFileDir     = "/"
)

func printConfigFile(dir string) {
	filePath := path.Join(dir, configFileName)
	b, err := afero.ReadFile(fs, filePath)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(b))
}

func TestDefaults(t *testing.T) {
	defer NewFs(t, []byte{}, DefaultConfigDir, afero.NewMemMapFs())()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)

	location := DefaultWindowLocation()
	assert.NoError(t, conf.Unmarshal(LocationKey, &location))
	assert.Equal(t, DefaultWindowLocation(), location)
}

func TestReadingConfigInDir(t *testing.T) {
	configBytes := FileToBytes(t, "./test-files/test.toml")
	defer NewFs(t, configBytes, testTomlFileDir, afero.NewMemMapFs())()
	conf, err := New(testTomlFileDir)
	require.NoError(t, err)

	var device Device
	deviceChanges := Device{}
	deviceChanges.ID = 789
	assert.NoError(t, conf.Unmarshal(DeviceKey, &device))
	assert.Equal(t, deviceChanges, device)

	windows := DefaultWindows()
	windowChanges := DefaultWindows()
	windowChanges.PowerOff = "+1s"
	assert.NoError(t, conf.Unmarshal(WindowsKey, &windows))
	assert.Equal(t, windowChanges, windows)

	var location Location
	locationChanges := Location{}
	locationChanges.Accuracy = 543
	assert.NoError(t, conf.Unmarshal(LocationKey, &location))
	assert.Equal(t, locationChanges, location)

	windowLocation := DefaultWindowLocation()
	windowLocationChanges := DefaultWindowLocation()
	windowLocationChanges.Accuracy = 543
	assert.NoError(t, conf.Unmarshal(LocationKey, &windowLocation))
	assert.Equal(t, windowLocationChanges, windowLocation)

	testHosts := DefaultTestHosts()
	testHostsChanges := DefaultTestHosts()
	testHostsChanges.PingRetries = 333
	assert.NoError(t, conf.Unmarshal(TestHostsKey, &testHosts))
	assert.Equal(t, testHostsChanges, testHosts)

	var battery Battery
	batteryChanges := Battery{}
	batteryChanges.NoBattery = 10
	assert.NoError(t, conf.Unmarshal(BatteryKey, &battery))
	assert.Equal(t, batteryChanges, battery)

	modemd := DefaultModemd()
	modemdChanges := DefaultModemd()
	modemdChanges.ConnectionTimeout = time.Second * 21
	assert.NoError(t, conf.Unmarshal(ModemdKey, &modemd))
	assert.Equal(t, modemdChanges, modemd)

	lepton := DefaultLepton()
	leptonChanges := DefaultLepton()
	leptonChanges.SPISpeed = 2
	assert.NoError(t, conf.Unmarshal(LeptonKey, &lepton))
	assert.Equal(t, leptonChanges, lepton)

	thermalRecorder := DefaultThermalRecorder()
	thermalRecorderChanges := DefaultThermalRecorder()
	thermalRecorderChanges.MaxSecs = 321
	assert.NoError(t, conf.Unmarshal(ThermalRecorderKey, &thermalRecorder))
	assert.Equal(t, thermalRecorderChanges, thermalRecorder)

	thermalThrottler := DefaultThermalThrottler()
	thermalThrottlerChanges := DefaultThermalThrottler()
	thermalThrottlerChanges.BucketSize = time.Second * 16
	assert.NoError(t, conf.Unmarshal(ThermalThrottlerKey, &thermalThrottler))
	assert.Equal(t, thermalThrottlerChanges, thermalThrottler)

	ports := DefaultPorts()
	portsChanges := DefaultPorts()
	portsChanges.Managementd = 3
	assert.NoError(t, conf.Unmarshal(PortsKey, &ports))
	assert.Equal(t, portsChanges, ports)

	var secrets Secrets
	secretsChanges := Secrets{}
	secretsChanges.DevicePassword = "pass"
	assert.NoError(t, conf.Unmarshal(SecretsKey, &secrets))
	assert.Equal(t, secretsChanges, secrets)

	thermalMotion := DefaultThermalMotion()
	thermalMotionChanges := DefaultThermalMotion()
	thermalMotionChanges.TempThresh = 398
	assert.NoError(t, conf.Unmarshal(ThermalMotionKey, &thermalMotion))
	assert.Equal(t, thermalMotionChanges, thermalMotion)
}

func TestWriting(t *testing.T) {
	defer NewFs(t, []byte{}, testTomlFileDir, afero.NewMemMapFs())()
	conf, err := New(testTomlFileDir)
	require.NoError(t, err)
	conf2, err := New(testTomlFileDir)
	require.NoError(t, err)

	d := randomDevice()
	w := randomWindows()
	l := randomLocation()
	h := randomTestHosts()
	require.NoError(t, conf.Set(DeviceKey, d))
	require.NoError(t, conf2.Set(WindowsKey, w))
	require.NoError(t, conf.Set(LocationKey, &l))
	require.NoError(t, conf2.Set(TestHostsKey, &h))

	conf, err = New(testTomlFileDir)
	require.NoError(t, err)
	d2 := Device{}
	require.NoError(t, conf.Unmarshal(DeviceKey, &d2))
	w2 := Windows{}
	require.NoError(t, conf.Unmarshal(WindowsKey, &w2))
	l2 := Location{}
	require.NoError(t, conf.Unmarshal(LocationKey, &l2))
	h2 := TestHosts{}
	require.NoError(t, conf.Unmarshal(TestHostsKey, &h2))

	require.Equal(t, d, d2)
	require.Equal(t, w, w2)
	require.Equal(t, l, l2)
	require.Equal(t, h, h2)
}

func TestFileLock(t *testing.T) {
	defer NewFs(t, []byte{}, testTomlFileDir, afero.NewMemMapFs())()
	lockTimeout = time.Millisecond * 100

	conf, err := New(testTomlFileDir)
	require.NoError(t, err)
	conf2, err := New(testTomlFileDir)
	require.NoError(t, err)

	require.NoError(t, conf.getFileLock())
	require.Equal(t, context.DeadlineExceeded, conf2.getFileLock())
	conf.fileLock.Unlock()
	require.NoError(t, conf2.getFileLock())
}

func TestSettingUpdated(t *testing.T) {
	defer NewFs(t, []byte{}, testTomlFileDir, afero.NewMemMapFs())()
	newNow()
	conf, err := New(testTomlFileDir)
	require.NoError(t, err)

	require.NoError(t, conf.Set(DeviceKey, randomDevice()))
	require.Equal(t, conf.Get(DeviceKey+".updated"), now())

	require.NoError(t, conf.Set(DeviceKey+".name", randString(10)))
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

func randomLocation() Location {
	return Location{
		Accuracy:  float32(randSrc.Int63()),
		Longitude: float32(randSrc.Int63()),
	}
}

func randomTestHosts() TestHosts {
	return TestHosts{
		URLs:         []string{randString(10), randString(20), randString(15)},
		PingRetries:  int(randSrc.Int63()),
		PingWaitTime: time.Duration(randSrc.Int63()) * time.Second,
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
