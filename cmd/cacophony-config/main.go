package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	config "github.com/TheCacophonyProject/go-config"
)

var version = "<not set>"

func main() {
	if err := runMain(); err != nil {
		log.Fatal(err)
	}
}

func runMain() error {
	log.SetFlags(0)
	log.Printf("running version: %s", version)

	settings, err := getNewSettings(os.Args[1:])
	if err != nil {
		return err
	}
	log.Printf("new settings: %+v", settings)

	conf, err := config.New(config.DefaultConfigDir)
	if err != nil {
		return err
	}

	for _, s := range settings {
		if err := conf.SetField(s.section, s.field, s.value); err != nil {
			return err
		}
	}

	return nil
}

type setting struct {
	section string
	field   string
	value   string
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
