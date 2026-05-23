package discdb

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"

	"github.com/hairyhenderson/go-which"
)

type Drive struct {
	Index  int
	Name   string
	Letter string
}

var makeMkVNotFound = errors.New("MakeMKV not found")

func findMakeMkvPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		const path = "C:\\Program Files (x86)\\MakeMKV\\makemkvcon64.exe"
		if exists(path) {
			return path, nil
		}
		return "", makeMkVNotFound
	case "darwin":
		const path = "/Applications/MakeMKV.app/Contents/MacOS/makemkvcon"
		if exists(path) {
			return path, nil
		}
		return "", makeMkVNotFound
	case "linux":
		var path = "/usr/bin/makemkvcon"
		if exists(path) {
			return path, nil
		}
		path = which.Which("makemkvcon")
		if exists(path) {
			return path, nil
		}
		return "", makeMkVNotFound
	default:
		var path = which.Which("makemkvcon")
		if exists(path) {
			return path, nil
		}
		return "", makeMkVNotFound
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

//public async Task WriteLogs(int driveIndex, string path, bool cleanLogs = true)
//        {
//            var info = new ProcessStartInfo
//            {
//                FileName = this.options.Value.Path,
//                Arguments = $"--robot --messages=\"{path}\" info disc:{driveIndex}"
//            };
//
//            await RunProcessAsync(info);
//
//            if (cleanLogs)
//            {
//                await Task.Delay(200); // wait for file handle to be released
//                try
//                {
//                    await CleanLogs(driveIndex, path);
//                }
//                catch (IOException e)
//                {
//                    throw new CleanLogFileException(path, e);
//                }
//            }
//        }

func WriteLogs(driveIndex int, output io.Writer) (err error) {
	makemkv, err := findMakeMkvPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(makemkv,
		"--robot",
		fmt.Sprintf(`--messages=-stdout`),
		"info",
		fmt.Sprintf("disc:%d", driveIndex),
	)
	slog.Debug("exec", "cmd", cmd.String())
	cmd.Stdout = output
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
