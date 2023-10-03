package data

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/NethermindEth/eigenlayer/internal/utils"
	"github.com/spf13/afero"
)

var backupFileNameRegex = regexp.MustCompile(`^(?P<instance_id>.*)-(?P<timestamp>[0-9]+)\.tar$`)

type Backup struct {
	id         string
	InstanceId string
	Timestamp  time.Time
	Version    string
	Commit     string
	Url        string
}

func (b *Backup) Id() string {
	if b.id == "" {
		h := sha1.Sum([]byte(fmt.Sprintf("%s-%d-%s-%s", b.InstanceId, b.Timestamp.Unix(), b.Version, b.Commit)))
		b.id = hex.EncodeToString(h[:])
	}
	return b.id
}

// BackupFromTar loads a backup information from a tar file.
func BackupFromTar(fs afero.Fs, src string) (*Backup, error) {
	// Check if file exists
	ok, err := afero.Exists(fs, src)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s", os.ErrNotExist, src)
	}
	// Check file name extension
	if ext := filepath.Ext(src); ext != ".tar" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidBackupName, src)
	}
	// Load state.json from tar
	instance, err := loadBackupTarStateJson(fs, src)
	if err != nil {
		return nil, err
	}
	// Load timestamp
	timestamp, err := loadBackupTarTimestamp(fs, src)
	if err != nil {
		return nil, err
	}
	return &Backup{
		InstanceId: instance.ID(),
		Timestamp:  timestamp,
		Version:    instance.Version,
		Commit:     instance.Commit,
		Url:        instance.URL,
	}, nil
}

// loadStateJsonFromTar loads the state.json file from a tar file.F
func loadBackupTarStateJson(fs afero.Fs, tarPath string) (*Instance, error) {
	// Open tar file
	tarFile, err := fs.OpenFile(tarPath, os.O_RDONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer tarFile.Close()
	// Load state.json
	stateData, err := utils.TarReadFile("data/state.json", tarFile)
	if err != nil {
		return nil, err
	}
	var instance Instance
	return &instance, json.Unmarshal(stateData, &instance)
}

func loadBackupTarTimestamp(fs afero.Fs, tarPath string) (time.Time, error) {
	// Open file
	tarFile, err := fs.OpenFile(tarPath, os.O_RDONLY, 0o644)
	if err != nil {
		return time.Time{}, err
	}
	defer tarFile.Close()
	// Load timestamp
	timestampData, err := utils.TarReadFile("timestamp", tarFile)
	if err != nil {
		return time.Time{}, err
	}
	timestampInt, err := strconv.ParseInt(string(timestampData), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(timestampInt, 0), nil
}

func ParseBackupName(backupName string) (instanceId string, timestamp time.Time, err error) {
	match := backupFileNameRegex.FindStringSubmatch(backupName)
	if len(match) != 3 {
		return "", time.Time{}, fmt.Errorf("%w: %s", ErrInvalidBackupName, backupName)
	}
	instanceId = match[1]
	timestampInt, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: %s", ErrInvalidBackupName, backupName)
	}
	timestamp = time.Unix(timestampInt, 0)
	return instanceId, timestamp, nil
}
