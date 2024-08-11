package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/averseabfun/logger"
	"github.com/fsnotify/fsnotify"
)

func WatchPath(path string, callback func(*fsnotify.Watcher)) *fsnotify.Watcher {
	logger.Logf(logger.LogDebug, "Creating watcher for path %s", path)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Logf(logger.LogFatal, "Attempted to watch path %s and got error %w", path, err)
	}
	go callback(watcher)

	logger.Logf(logger.LogDebug, "Adding path %s to watcher", path)
	err = watcher.Add(path)
	if err != nil {
		logger.Logf(logger.LogFatal, "Attempted to watch path %s and got error %w", path, err)
	}

	return watcher
}

var current_line = 0
var base_path = "./game"

const input_name = "-99 input"
const input_line = 99

func WriteLine(line string) {
	os.RemoveAll(filepath.Join(base_path, "-"+fmt.Sprintf("%02d", current_line)+" "+line))
	os.MkdirAll(filepath.Join(base_path, "-"+fmt.Sprintf("%02d", current_line)+" "+line), 0770)
	current_line++
}

func RemoveLine(line int) error {
	matches, err := filepath.Glob(filepath.Join(base_path, "-"+fmt.Sprintf("%02d", line)+" *"))
	logger.Logf(logger.LogDebug, "Line: %d Matches: %v", line, matches)
	if err != nil {
		return err
	}
	matches2, err := filepath.Glob(filepath.Join(base_path, "history", "-"+fmt.Sprintf("%02d", line)+" *"))
	if err != nil {
		return err
	}
	var from = matches[0]
	if len(matches2) > 0 {
		line++
		matches[0] = filepath.Join(base_path, "-"+fmt.Sprintf("%02d", line)+" "+strings.SplitN(matches[0], " ", 2)[1])
	}

	err = os.Rename(from, filepath.Join(base_path, "history", filepath.Base(matches[0])))
	if err != nil {
		logger.Logf(logger.LogFatal, "Failed to remove %s: %v", matches[0], err)
	} else {
		logger.Logf(logger.LogDebug, "Removed %s", matches[0])
	}

	if line == current_line-1 {
		current_line--
	}
	return nil
}

func MoveAllUp() error {
	matches, err := filepath.Glob(filepath.Join(base_path, "-*"))
	if err != nil {
		return err
	}

	for _, match := range matches {
		if match == input_name {
			continue
		}
		original_match := match
		match = strings.TrimPrefix(filepath.Base(match), "-")
		rest := string(match[0:2])
		match = string(match[2:])
		var line, err = strconv.Atoi(rest)
		if err != nil {
			return err
		}
		if line == input_line {
			continue
		}
		if line == 0 {
			RemoveLine(line)
			continue
		}
		line--

		err = os.Rename(original_match, filepath.Join(base_path, "-"+fmt.Sprintf("%02d", line)+""+match))
		if err != nil {
			logger.Logf(logger.LogFatal, "Failed to remove %s: %v", match, err)
		} else {
			logger.Logf(logger.LogDebug, "Removed %s", match)
		}
	}
	current_line--
	return nil
}

var nextCreateIsInput = false

func main() {
	var input = make(chan string)
	os.RemoveAll(base_path)
	os.MkdirAll(filepath.Join(base_path, input_name), 0770)
	os.MkdirAll(filepath.Join(base_path, "history"), 0770)
	defer WatchPath(base_path, func(watcher *fsnotify.Watcher) {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				logger.Logf(logger.LogDebug, "event:", event)
				if event.Has(fsnotify.Write) {
					logger.Logf(logger.LogDebug, "modified file:", event.Name)
				}
				if event.Has(fsnotify.Rename) && filepath.Base(event.Name) == input_name {
					nextCreateIsInput = true
					os.MkdirAll(filepath.Join(base_path, input_name), 0770)
					continue
				}
				if event.Has(fsnotify.Create) && nextCreateIsInput && !strings.HasPrefix(filepath.Base(event.Name), "-") {
					nextCreateIsInput = false
					os.Remove(event.Name)
					input <- filepath.Base(event.Name)
					continue
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Logf(logger.LogDebug, "error:", err)
			}
		}
	}).Close()
	WriteLine("test")
	for inp := range input {
		logger.Logf(logger.LogDebug, "got input %s", inp)
		if inp == "test" {
			WriteLine("This is working!!")
		} else if inp == "remove" {
			RemoveLine(current_line - 1)
		} else if inp == "move" {
			MoveAllUp()
		}
	}
}
