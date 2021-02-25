package paths

import (
	"fmt"
	"github.com/l3uddz/tqm/logger"
	"os"
	"path/filepath"
	"time"
)

/* Structs */

type Path struct {
	Path         string
	RealPath     string
	FileName     string
	Directory    string
	IsDir        bool
	Size         int64
	ModifiedTime time.Time
}

/* Types */

type callbackAllowed func(string) *string

/* Vars */

var (
	log = logger.GetLogger("pathutils")
)

/* Public */

func GetPathsInFolder(folder string, includeFiles bool, includeFolders bool, acceptFn callbackAllowed) ([]Path, uint64) {
	var paths []Path
	var size uint64 = 0

	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk func: %w", err)
		}

		// skip files if not wanted
		if !includeFiles && !info.IsDir() {
			log.Tracef("Skipping file: %s", path)
			return nil
		}

		// skip folders if not wanted
		if !includeFolders && info.IsDir() {
			log.Tracef("Skipping folder: %s", path)
			return nil
		}

		// skip paths rejected by accept callback
		realPath := path
		finalPath := path
		if acceptFn != nil {
			if acceptedPath := acceptFn(path); acceptedPath == nil {
				log.Tracef("Skipping rejected path: %s", path)
				return nil
			} else {
				finalPath = *acceptedPath
			}
		}

		foundPath := Path{
			Path:         finalPath,
			RealPath:     realPath,
			FileName:     info.Name(),
			Directory:    filepath.Dir(path),
			IsDir:        info.IsDir(),
			Size:         info.Size(),
			ModifiedTime: info.ModTime(),
		}

		paths = append(paths, foundPath)
		size += uint64(info.Size())

		return nil

	})

	if err != nil {
		log.WithError(err).Errorf("Failed to retrieve paths from %s", folder)
	}

	return paths, size
}
