package files

import (
	"fmt"
	"os"
	"path"

	"revengy.io/gco/agent/internal/hash"
	"revengy.io/gco/agent/internal/log"
)

// directory specifies the directory to use when writing configuration files.
var directory string

func SetDirectory(dir string) {
	log.Debugf("Setting configuration directory=%s", dir)
	directory = dir
}

// GetDirectory returns the directory for storing configuration files or defaults to '/etc/gco' when not configured.
func GetDirectory() (string, error) {
	dir := directory
	if dir == "" {
		dir = "/tmp/gco"
	}

	// create directory structure if it does not exist
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return "", fmt.Errorf("failed creating config dir: %v", err)
	}

	return dir, nil
}

// WriteConfigFileFromString writes the given content to a configuration file with the given name.
// It will return the absolute path of the written file if successful.
func WriteConfigFileFromString(content string, name string) (string, error) {
	hashed := hash.CalculateHashFromString(content)
	short := hash.ShortHash(hashed)

	dir, err := GetDirectory()
	if err != nil {
		return "", err
	}

	// write content to location
	location := path.Join(dir, fmt.Sprintf("%s.%s", short, name))

	// TODO: check if file already exists -> no need to write new file

	err = os.WriteFile(location, []byte(content), 0644)
	if err != nil {
		return "", err
	}

	log.Debugf("Wrote configuration file for name=%s with hash=%s", name, short)
	return location, nil
}

// RemoveConfigFile tries to remove the configuration file from the specified location.
func RemoveConfigFile(file string) {
	err := os.Remove(file)
	if err != nil {
		log.Warnf("Failed to remove file=%s: %v", file, err)
	}
}
