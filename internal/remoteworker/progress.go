package remoteworker

import (
	"fmt"
	"os"
	"path/filepath"
)

const uploadBufSize = 4 * 1024 * 1024

// UploadProgress reports bytes transferred during SFTP uploads.
type UploadProgress func(done, total int64, label string)

func uploadProgressStep(lastPct *int, done, total int64, label string, onLog func(string), onProgress UploadProgress) {
	if total <= 0 {
		return
	}
	pct := int(done * 100 / total)
	if pct > 100 {
		pct = 100
	}
	if *lastPct >= 0 && pct-*lastPct < 5 && done < total {
		return
	}
	*lastPct = pct
	if onLog != nil {
		onLog(fmt.Sprintf("%s: %d%% (%s / %s)", label, pct, formatSize(done), formatSize(total)))
	}
	if onProgress != nil {
		onProgress(done, total, label)
	}
}

func dirByteSize(root string) (int64, error) {
	var total int64
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}
