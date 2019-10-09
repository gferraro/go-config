package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	config "github.com/TheCacophonyProject/go-config"
	"github.com/alexflint/go-arg"
)

var version = "<not set>"

type Args struct {
	ConfigDir string   `arg:"-c,--config" help:"path to configuration directory"`
	Write     bool     `arg:"-w,--write" help:"write to config file"`
	Read      bool     `arg:"-r,--read" help:"read from the config file"`
	Input     []string `arg:"positional"`
}

func (Args) Version() string {
	return version
}

func procArgs() Args {
	var args Args
	args.ConfigDir = config.DefaultConfigDir
	arg.MustParse(&args)
	return args
}

func main() {
	if err := runMain(); err != nil {
		log.Fatal(err)
	}
}

func runMain() error {
	args := procArgs()
	log.SetFlags(0)
	log.Printf("running version: %s", version)

	if args.Write {
		return writeNewSettings(&args)
	}
	if args.Read {
		return readConfig(&args)
	}
	return errors.New("no valid arguments given")
}

func readConfig(args *Args) error {
	conf, err := config.New(args.ConfigDir)
	if err != nil {
		return err
	}

	for _, section := range args.Input {
		var m map[string]interface{}
		if err := conf.Unmarshal(section, &m); err != nil {
			return err
		}
		log.Printf("section: '%s', values: '%s'", section, m)
	}
	return nil
}

type setting struct {
	section string
	field   string
	value   string
}

func writeNewSettings(args *Args) error {
	settings, err := getNewSettings(args.Input)
	if err != nil {
		return err
	}
	log.Printf("new settings: %+v", settings)

	conf, err := config.New(args.ConfigDir)
	if err != nil {
		return err
	}
	conf.AutoWrite = false // Only write if there were no errors in writing all the settings

	sections := map[string]int{}
	for _, s := range settings {
		if err := conf.SetField(s.section, s.field, s.value); err != nil {
			return err
		}
		if _, ok := sections[s.section]; !ok {
			sections[s.section] = 0
		}
	}
	if err := conf.Write(); err != nil {
		return err
	}

	for section, _ := range sections {
		raw := map[string]interface{}{}
		if err := conf.Unmarshal(section, &raw); err != nil {
			return err
		}
		log.Printf("section: '%s', values: '%s'", section, raw)
	}

	return nil
}

func getNewSettings(args []string) ([]setting, error) {
	settings := []setting{}
	for _, arg := range args {
		spl := strings.Split(arg, "=")
		if len(spl) != 2 {
			return nil, fmt.Errorf("'%s' should contain one '='", arg)
		}
		key := spl[0]
		val := spl[1]

		spl = strings.Split(key, ".")
		if len(spl) != 2 {
			return nil, fmt.Errorf("'%s' should contain one '.' Nested fields are not supported yet", arg)
		}
		settings = append(settings, setting{
			section: spl[0],
			field:   spl[1],
			value:   val,
		})
	}
	return settings, nil
}
