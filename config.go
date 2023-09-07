package main

import (
	"crypto/rand"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

var config *Config
var previousMemorySize int
var memoryBuffer []byte

func init() {
	config = &Config{}
	memoryBuffer = make([]byte, 0)
}

type CPUConfig struct {
	MinPrimesCalclated int `yaml:"minPrimesCalculated"`
	MaxPrimesCalclated int `yaml:"maxPrimesCalculated"`
}

type MemoryConfig struct {
	Usage int `yaml:"usage"`
}

type Config struct {
	ErrorRate int          `yaml:"errorRate"`
	CPU       CPUConfig    `yaml:"cpu"`
	Memory    MemoryConfig `yaml:"memory"`
}

func reloadConfig(path string) error {
	fmt.Printf("reloading config %s...\n", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if config.CPU.MaxPrimesCalclated <= config.CPU.MinPrimesCalclated {
		return fmt.Errorf("max primes calculated must be greater than min primes calculated")
	}

	if config.Memory.Usage != previousMemorySize {
		makeMemoryBuffer()
		previousMemorySize = config.Memory.Usage
	}
	fmt.Printf("config is now: %+v\n", config)
	return nil
}

func makeMemoryBuffer() {
	fmt.Printf("allocating %d bytes\n", config.Memory.Usage)
	memoryBuffer = make([]byte, config.Memory.Usage)
	rand.Read(memoryBuffer)
}

func LoadConfig(path string) (*fsnotify.Watcher, error) {
	err := reloadConfig(path)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	// Start listening for events.
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op == fsnotify.Remove {
					err := watcher.Remove(event.Name)
					if err != nil {
						log.Println("failed to remove watcher:", err)
					}
					err = watcher.Add(path)
					if err != nil {
						log.Println("failed to create watcher:", err)
					}
					err = reloadConfig(path)
					if err != nil {
						log.Println("config file reload error:", err)
					}
				}
				if event.Op == fsnotify.Write {
					err := reloadConfig(path)
					if err != nil {
						log.Println("config file reload error:", err)
					}
				}
			case err := <-watcher.Errors:
				if err != nil {
					log.Println("config file watch error:", err)
				}
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		return nil, fmt.Errorf("failed to start watching %s: %w", path, err)
	}
	return watcher, nil
}
