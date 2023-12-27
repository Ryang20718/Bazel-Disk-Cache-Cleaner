package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	BazelCacheDir string
	KeepFilesAccessedDays int
)

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Short: "Script to clean bazel cache",
	}
}
/*
Bazel is a hungry process that lacks any sort of cache bounding mechanism.

The ticket tracking that progress https://github.com/bazelbuild/bazel/issues/5139 has been opened since 2018.
Since we don't know when that will be implemented, this script is intended to workaround that.
It leverages access time to keep the bazel cache bounded by deleting all files greater than the atime specified.
*/

func main() {
	rootCmd := NewRootCmd()
	rootCmd.PersistentFlags().StringVar(&BazelCacheDir, "bazel-cache-dir", "", "path to bazel cache directory to clear")
	rootCmd.PersistentFlags().IntVar(&KeepFilesAccessedDays, "keep-files-access-days", 0, "purge files with access time greater than")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "clean",
		Short: "clean bazel cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Printf("Starting Clean up Process of Bazel Directory %s. This may take a couple of minutes", BazelCacheDir)
			var fileList []string
			err := filepath.Walk(BazelCacheDir, func(path string, info os.FileInfo, err error) error {
				// Get the syscall.Stat_t structure
				stat := info.Sys().(*syscall.Stat_t)
				var accessTime time.Time
				atimField := reflect.ValueOf(stat).FieldByName("Atim")
				atimespecField := reflect.ValueOf(stat).FieldByName("Atimespec")
				if atimField.IsValid() { // Valid for linux
					secField := atimField.FieldByName("Sec")
					nsecField := atimField.FieldByName("Nsec")
					if secField.IsValid() && nsecField.IsValid() {
						accessTime = time.Unix(int64(secField.Uint()), int64(nsecField.Uint()))
					}
				}else if atimespecField.IsValid() { // Macos
					secField := atimespecField.FieldByName("Sec")
					nsecField := atimespecField.FieldByName("Nsec")
					if secField.IsValid() && nsecField.IsValid() {
						accessTime = time.Unix(int64(secField.Uint()), int64(nsecField.Uint()))
					}
				}
				timeKeepFilesAccessedDays := time.Duration(KeepFilesAccessedDays) * 24 * time.Hour
				if accessTime.Add(timeKeepFilesAccessedDays).Before(time.Now()) && path != BazelCacheDir && !info.IsDir() {
					// Check if Access time is greater than desired days to keep
					// don't delete cache dir, otherwise you'll have to purge bazel cache completely
					fileList = append(fileList, path)
				}
				if err != nil {
					return fmt.Errorf("failing to walk path %s err: %v", BazelCacheDir, err)
				}
				return nil
			})
			if err != nil {
				return err
			}
			for _, file := range fileList {
				err := os.RemoveAll(file)
				if err != nil {
					return fmt.Errorf("failing to remove file err: %v", err)
				}
			}
			log.Println("Finished Cleaning Bazel Cache up!")
			return nil
		},
	})
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
