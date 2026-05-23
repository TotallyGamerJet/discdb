package main

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/TotallyGamerJet/discdb"
)

func run() (err error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	discdbPath := filepath.Join(cache, "discdb")
	fmt.Println(discdbPath)
	err = discdb.Download(discdbPath)
	if err != nil {
		return err
	}
	output, err := os.Create("out.txt")
	if err != nil {
		return err
	}
	defer func(output *os.File) {
		err = errors.Join(err, output.Close())
	}(output)
	fmt.Println(discdb.WriteLogs(0, output))
	return nil
}

func main() {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	slog.SetDefault(slog.New(h))
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
