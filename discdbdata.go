package discdb

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
)

type slogWriter struct {
	*slog.Logger
}

func (a slogWriter) Write(p []byte) (n int, err error) {
	output := string(p)
	for _, v := range strings.Split(output, "\r") {
		if len(v) == 0 {
			continue
		}
		slog.Debug(v)
	}
	return len(p), nil
}

const repoURL = "https://github.com/TheDiscDb/data"

// Download the latest version of the discdb data
func Download(outputPath string) error {
	progressor := slogWriter{slog.Default()}
	repo, err := git.PlainClone(outputPath, &git.CloneOptions{
		URL:      repoURL,
		Progress: progressor,
	})

	if err != nil {
		// If repo already exists, open it instead
		if !errors.Is(err, git.ErrTargetDirNotEmpty) && !os.IsExist(err) {
			return err
		}
		repo, err = git.PlainOpen(outputPath)
		if err != nil {
			return err
		}
	}

	// Fetch latest changes
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Progress:   progressor,
	})

	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}

	// Update working tree (simple "pull" behavior)
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   progressor,
	})

	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}

	return nil
}

type LogLine struct {
	OriginalLine string
}

type HashInfoLogLine struct {
	LogLine
	Index        int
	Name         string
	CreationTime time.Time
	Size         int64
}

func ParseHashInfoLogLine(line string) (logLine HashInfoLogLine, err error) {
	//string[] parts = line.Substring(4).Split(',');
	//
	//        var result = new HashInfoLogLine
	//        {
	//            OriginalLine = line,
	//            Index = TryParseInt(0, parts),
	//            Name = GetString(1, parts),
	//            CreationTime = TryParseDateTime(2, parts),
	//            Size = TryParseLong(3, parts)
	//        };
	//
	//        return result;
	parts := strings.Split(line[4:], ",")
	if len(parts) != 4 {
		return logLine, errors.New("discdb: invalid hash info log line")
	}
	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return logLine, err
	}
	size, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return logLine, err
	}
	return HashInfoLogLine{
		LogLine:      LogLine{OriginalLine: line},
		Index:        index,
		Name:         parts[1],
		CreationTime: time.Now(),
		Size:         size,
	}, nil
}

type DiscHashInfo struct {
	Hash  string
	Files []FileHashInfo
}

type FileHashInfo struct {
	Index        int
	Name         string
	CreationTime time.Time
	Size         int64
}

const LinePrefix = "HSH"

func HashLogFile(log io.Reader) (info DiscHashInfo, err error) {
	//foreach (var line in lines)
	//        {
	//            if (line.StartsWith(HashInfoLogLine.LinePrefix, StringComparison.OrdinalIgnoreCase))
	//            {
	//                var hashInfo = HashInfoLogLine.Parse(line);
	//                info.Files.Add(new FileHashInfo
	//                {
	//                    Index = hashInfo.Index,
	//                    Name = hashInfo.Name,
	//                    CreationTime = hashInfo.CreationTime,
	//                    Size = hashInfo.Size
	//                });
	//            }
	//        }
	//
	//        info.Hash = info.Files.CalculateHash();
	for lines := range lines(log) {
		if !strings.HasPrefix(lines, LinePrefix) {
			continue
		}

	}
	return info, err
}

func HashMediaFS(fsys fs.FS) (string, error) {
	var mediaPath string
	pattern := "*"

	switch {
	case existsFS(fsys, "BDMV/STREAM"):
		mediaPath = "BDMV/STREAM"
		pattern = "*.m2ts"
	case existsFS(fsys, "VIDEO_TS"):
		mediaPath = "VIDEO_TS"
		pattern = "*"
	default:
		return "", errors.New("discdb: unknown media disc")
	}

	entries, err := fs.ReadDir(fsys, mediaPath)
	if err != nil {
		return "", fmt.Errorf("discdb: failed to read directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return "", fmt.Errorf("discdb: failed to match pattern: %w", err)
		}
		if matched {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names) // Order files by name

	var fileSizes []int64
	for _, name := range names {
		info, err := fs.Stat(fsys, mediaPath+"/"+name)
		if err != nil {
			continue
		}
		fileSizes = append(fileSizes, info.Size())
	}

	return hashFromFileSizes(fileSizes)
}

// hashFromFileSizes takes a list of file sizes sorted by filename and returns a hash.
func hashFromFileSizes(fileSizes []int64) (string, error) {
	hash, err := calculateHash(fileSizes)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(hash), nil
}

// calculateHash takes a list of file's sizes sorted by the filename and returns a hash of them.
func calculateHash(filesSizes []int64) (string, error) {
	hash := md5.New()

	for _, file := range filesSizes {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(file))
		hash.Write(buf)
	}

	sum := hash.Sum(nil)
	if len(sum) == 0 {
		return "", errors.New("discdb: unable to create disc hash")
	}

	return hex.EncodeToString(sum), nil
}

func existsFS(fsys fs.FS, path string) bool {
	info, err := fs.Stat(fsys, path)
	return err == nil && info.IsDir()
}

// lines returns a range-over-func iterator over lines in r.
func lines(r io.Reader) iter.Seq[string] {
	return func(yield func(string) bool) {
		sc := bufio.NewScanner(r)

		// Optional: allow longer lines than 64K
		sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		for sc.Scan() {
			if !yield(sc.Text()) {
				return
			}
		}

		// You may want to handle sc.Err() in real code
	}
}
