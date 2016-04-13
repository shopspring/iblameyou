package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mwek/iblameyou/internal"
	"gopkg.in/yaml.v2"
)

var config = flag.String("config",
	os.Getenv("HOME")+"/.iblameyou.yaml",
	"path to configuration file")

type Config struct {
	Format internal.Format
	Source internal.Source
}

func DefaultConfig() (c Config) {
	c.Format.Colors = internal.DefaultPalette()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Stderr = ioutil.Discard
	if repo, err := cmd.Output(); err == nil {
		c.Source.Repository = strings.TrimSpace(string(repo))
	}
	return c
}

func main() {
	flag.Parse()

	cfg := DefaultConfig()
	if b, err := ioutil.ReadFile(*config); err == nil {
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			log.Fatalf("Failed to unmarshal config:\n%s", err)
		}
	}

	if cfg.Source.Repository == "" {
		log.Fatal("Repository not provided and not in a Git repository.")
	}

	ui := internal.UI{}
	err := ui.Init(&cfg.Format)
	if err != nil {
		log.Fatalf("Failed to initialize the UI:\n%s", err)
	}
	defer ui.Close()

	go func() {
		dump, err := cfg.Source.ParseDump(os.Stdin)
		if err != nil {
			log.Fatalf("Failed to parse dump:\n%s", err)
		}
		ui.RenderDump(dump)
	}()

	ui.Loop()
}
