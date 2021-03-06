package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/SharperShape/vanadia/markdown"
	"github.com/bukalapak/snowboard/api"
	"os"
	"path"

	"github.com/SharperShape/vanadia/blueprint"
	"github.com/SharperShape/vanadia/config"
	"github.com/SharperShape/vanadia/postman"
)

const (
	defaultConfigFileName = "vanadia.yml"
)

func main() {
	var (
		inFileName     = flag.String("input", "", "Location of .apib file as input.")
		outFileName    = flag.String("output", "", "Location of Postman file.")
		configFileName = flag.String("config", defaultConfigFileName, "Location of vanadia.yml.")
		printVersion   = flag.Bool("version", false, "Display Vanadia version")

		inFileByte []byte
		outFile    *os.File
		err        error
		version    string
	)

	// Short version for version (pun non intended)
	flag.BoolVar(printVersion, "v", false, "Display Vanadia version")

	flag.Parse()

	if version == "" {
		version = "HEAD"
	}
	if *printVersion {
		fmt.Println("Vanadia version:", version)
		return
	}

	var file *os.File
	if *inFileName == "" {
		wd, err := os.Getwd()
		if err != nil {
			inFileByte, err = markdown.Preprocess(os.Stdin, wd)
		}
	} else if file, err = os.Open(*inFileName); err == nil {
		inFileByte, err = markdown.Preprocess(bufio.NewReader(file), path.Dir(file.Name()))
		file.Close()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading input:", err)
		os.Exit(1)
	}

	if *outFileName == "" {
		outFile = os.Stdout
	} else {
		outFile, err = os.Create(*outFileName)
		if err != nil {
			os.Exit(1)
		}
		defer outFile.Close()
	}

	cfg, err := config.FromFile(*configFileName)
	if err != nil {
		func() {
			switch err.(type) {
			case *os.PathError:
				if *configFileName == defaultConfigFileName {
					// If no default config file is defined, then we just
					// use default Config value
					cfg = config.DefaultConfig
					return
				} else {
					fmt.Fprintln(os.Stderr, "error reading config:", err)
					os.Exit(1)
				}
			}
			fmt.Fprintln(os.Stderr, "error reading config:", err)
			os.Exit(1)
		}()
	}

	bp, err := blueprint.GetStructure(inFileByte)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error parsing blueprint:", err)
		os.Exit(1)
	}

	collection, err := postman.CreateCollection(bp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating collection:", err)
		os.Exit(1)
	}

	if cfg.PostmanHost.Enabled {
		postmanHost := getPostmanHost(bp.Metadata)
		if postmanHost != "" {
			postman.HostToEnvFixed(&collection, postmanHost)
		}
	}
	if cfg.SchemeToEnv.Enabled {
		postman.SchemeToEnv(&collection, cfg.SchemeToEnv.Name)
	}
	if cfg.HostToEnv.Enabled && cfg.HostToEnv.Segments > 0 {
		postman.HostToEnv(&collection, cfg.HostToEnv.Segments, cfg.HostToEnv.Name)
	}
	if cfg.AuthTokenToEnv.Enabled {
		postman.AuthTokenToEnv(&collection, cfg.AuthTokenToEnv.Name)
	}
	postman.AddGlobalHeaders(&collection, cfg.GlobalHeaders)

	json, err := json.MarshalIndent(collection, "", "\t")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error exporting to JSON:", err)
		os.Exit(1)
	}

	_, err = outFile.Write(json)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error on write output:", err)
		os.Exit(1)
	}
}

func getPostmanHost(meta []api.Metadata) string {
	for _, entry := range meta {
		if entry.Key == "POSTMAN_HOST" {
			return entry.Value
		}
	}
	return ""
}
