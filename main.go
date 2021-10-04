package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// TODO: Deleting a file does not trigger a change

var errMatchFound = errors.New("Match found, exiting.")

func parseIgnore(ignoreString string) map[string]struct{} {
	splitIgnoreString := strings.Split(ignoreString, ",")
	var ignores = make(map[string]struct{})

	for _, ignore := range splitIgnoreString {
		ignores[ignore] = struct{}{}
	}

	return ignores
}

func walkDir(absDir, globPattern, ignore string, lastCheck time.Time) error {
	fileSystem := os.DirFS(absDir)
	ignores := parseIgnore(ignore)

	return fs.WalkDir(fileSystem, ".", func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		_, shouldIgnore := ignores[dirEntry.Name()]

		if dirEntry.IsDir() {
			if shouldIgnore {
				return fs.SkipDir
			}
			return nil
		} else {
			if shouldIgnore {
				return nil
			}

			match, err := filepath.Match(globPattern, dirEntry.Name())
			if err != nil {
				return err
			}

			if match {
				fileInfo, err := dirEntry.Info()
				if err != nil {
					return err
				}

				if fileInfo.ModTime().After(lastCheck) {
					fmt.Println(dirEntry.Name(), "changed. Reloading..")
					return errMatchFound
				}
			}
		}
		return nil
	})
}

func shouldReload(absDir, globPattern, ignore string, lastCheck time.Time) (bool, error) {
	err := walkDir(absDir, globPattern, ignore, lastCheck)
	if errors.Is(errMatchFound, err) {
		return true, nil
	} else if err != nil {
		return false, err
	}

	return false, nil
}

func reloadCommand(cmdString string) (*os.Process, error) {
	cmd := exec.Command("/bin/sh", "-c", cmdString)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd.Process, nil
}

func stopProcess(process *os.Process) error {
	if *process == (os.Process{}) {
		fmt.Println("Empty process, nothing to kill.")
		return nil
	}
	err := process.Signal(syscall.SIGTERM)
	return err
}

func reload(absDir, glob, ignore, cmd string, pollRate time.Duration) {
	tickChan := time.Tick(pollRate * time.Millisecond)
	lastCheck := time.Time{}
	isReloading := false
	process := &os.Process{}

	for _ = range tickChan {
		var err error
		isReloading, err = shouldReload(absDir, glob, ignore, lastCheck)
		if err != nil {
			log.Fatal("Error scanning files: ", err)
		}
		lastCheck = time.Now()

		if isReloading {
			err := stopProcess(process)
			if err != nil {
				log.Fatal("stopProcess : ", err)
			}
			process.Wait()
			process, err = reloadCommand(cmd)
			if err != nil {
				log.Fatal("reloadCommand : ", err)
			}
			isReloading = false
		}
	}
}

func main() {
	dir := flag.String("dir", "", "Directory to recursively watch for changes.")
	cmd := flag.String("cmd", "", "Command to run when changes are detected.")
	glob := flag.String("glob", "*.go", "Glob to match filenames against.")
	ignore := flag.String("ignore", ".git", "Comma-deliminated list of files and directories to ignore.")
	pollRate := flag.Duration("poll-rate", 100, "Time in milliseconds to wait between checks for changes.")
	flag.Parse()

	if *cmd == "" {
		log.Fatal("--cmd must be supplied.")
	}

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatal("Error parsing --dir: ", err)
	}

	reload(absDir, *glob, *ignore, *cmd, *pollRate)
}
