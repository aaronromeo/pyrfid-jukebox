package helper

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
)

const DefaultASLAConfig = "/home/pi/.asoundrc"
const DefaultProjectRoot = "/home/pi/workspace/pyrfid-jukebox"
const DefaultRelativeASLAConfig = "/system/home/.asoundrc"

func (r *RealALSAConfigUpdater) UpdateALSAConfig(cmdExecutor CommandExecutor) error {
	if hasChanged, errHC := HasALSAConfigChanged(); errHC != nil {
		return errHC
	} else if hasChanged {
		log.Println("ALSA config has changed. Copying over system config...")

		aslaConfig := GetALSASystemConfig()
		projectAslaConfig := GetALSARepoConfig()

		if err := CopyFile(aslaConfig, projectAslaConfig); err != nil {
			log.Printf("Error copying file: %v", err)
			return err
		}

		if err := cmdExecutor.Command("sudo", "alsactl", "restore").Run(); err != nil {
			log.Printf("Error executing alsactl restore: %v", err)
			return err
		}
	}
	return nil
}

func HasALSAConfigChanged() (bool, error) {
	aslaConfig := GetALSASystemConfig()
	projectAslaConfig := GetALSARepoConfig()

	var diff bool
	var err error
	if diff, err = FilesAreDifferent(aslaConfig, projectAslaConfig); err != nil {
		return false, err
	}
	return diff, nil
}

func CopyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0600)
	if err != nil {
		return err
	}

	return nil
}

func GetALSASystemConfig() string {
	aslaConfig := os.Getenv("PJ_ALSA_CONFIG")
	if aslaConfig == "" {
		aslaConfig = DefaultASLAConfig
	}
	return aslaConfig
}

func GetALSARepoConfig() string {
	projectRoot := os.Getenv("PJ_PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = DefaultProjectRoot
	}
	return filepath.Join(projectRoot, DefaultRelativeASLAConfig)
}

func FilesAreDifferent(file1, file2 string) (bool, error) {
	bytes1, err := os.ReadFile(file1)
	if err != nil {
		return false, err
	}

	bytes2, err := os.ReadFile(file2)
	if err != nil {
		return false, err
	}

	return !bytes.Equal(bytes1, bytes2), nil
}
