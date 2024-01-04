package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

/*
Bazel is a hungry process that lacks any sort of cache bounding mechanism.

The ticket tracking that progress https://github.com/bazelbuild/bazel/issues/5139 has been opened since 2018.
Since we don't know when that will be implemented, this script is intended to workaround that.
It leverages access time to keep the bazel cache bounded by deleting all files greater than the atime specified.
*/

var (
	BazelCacheDir          string
	KeepFilesAccessedDays  int
	ExternalRepoTargetList string
	Verbose                bool
)

type CleanBazelInput struct {
	BazelCacheDir             string
	KeepFilesAccessedDays     int
	ActiveExternalRepoTargets map[string]string
	BlackListFiles            map[string]string
	BlackListDirectories      map[string]string
}

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Short: "Script to clean bazel cache",
	}
}

/*
Generate a mapping to what targets still are "active" in the bazel cache under external repos

Each target in the external repo is structured as <target> accompanied by @<target>.marker
We need to rid both of these in order to fetch if they were incorrectly removed
*/
func generateExternalRepoTargets(externalRepoTargetList string) (map[string]string, error) {
	activeExternalRepoTargets := make(map[string]string) // ideally we have a set....
	file, err := os.Open(externalRepoTargetList)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		targetName := scanner.Text()
		markerName := fmt.Sprintf("@%s.marker", targetName)
		activeExternalRepoTargets[targetName] = ""
		activeExternalRepoTargets[markerName] = ""
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return activeExternalRepoTargets, nil
}

func createLogger(verbose bool) *zap.SugaredLogger {
	logger, _ := zap.NewProduction()
	if verbose {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync() //nolint:errcheck
	log := logger.Sugar()

	return log
}

// Check if the directory or file is blacklisted
func isBlacklisted(path string, f os.FileInfo, input CleanBazelInput) bool {
	_, blackListedFile := input.BlackListFiles[f.Name()]
	_, blackListedDir := input.BlackListDirectories[filepath.Base(path)]
	return blackListedFile || blackListedDir
}

// Check if the directory we're trying to remove will break bazel and cause us to purge cache
func isBazelDirectory(path string, input CleanBazelInput) bool {
	bazelJvmToolDir := strings.Contains(filepath.Dir(path), "embedded_tools")
	bazelInstallDir := strings.Contains(filepath.Dir(path), "install")
	bazelInvocationDir := filepath.Base(path) != "cache" && filepath.Dir(path) == input.BazelCacheDir
	return bazelJvmToolDir || bazelInstallDir || bazelInvocationDir
}

/*
Find files to remove that have an access time greater than input.timeKeepFilesAccessedDays

Skip any directories with "install" in the path
*/
func findFilesToClean(input CleanBazelInput, log *zap.SugaredLogger) ([]string, error) {
	filesToRemove := []string{}
	err := filepath.Walk(BazelCacheDir, func(path string, f os.FileInfo, err error) error {
		// Get the syscall.Stat_t structure
		stat := f.Sys().(*syscall.Stat_t)
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
		timeKeepFilesAccessedDays := time.Duration(input.KeepFilesAccessedDays) * 24 * time.Hour
		if accessTime.Add(timeKeepFilesAccessedDays).Before(time.Now()) && path != input.BazelCacheDir {
			// Check if Access time is greater than desired days to keep

			_, activeExternalTarget := input.ActiveExternalRepoTargets[f.Name()]
			activeTarget := f.IsDir() && activeExternalTarget
			if activeTarget {
				log.Debugf("skipping path %s active Target", path)
				return filepath.SkipDir
			}

			if !activeExternalTarget && !isBlacklisted(path, f, input) && !isBazelDirectory(path, input) {
				log.Debugf("adding file to remove %s", path)
				filesToRemove = append(filesToRemove, path)
			} else {
				log.Debugf("skipping path %s", path)
			}
		}
		if err != nil {
			return fmt.Errorf("failing to walk path %s err: %v", BazelCacheDir, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filesToRemove, nil
}

func main() {
	rootCmd := NewRootCmd()
	rootCmd.PersistentFlags().StringVar(&BazelCacheDir, "bazel-cache-dir", "", "path to bazel cache directory to clear")
	rootCmd.PersistentFlags().IntVar(&KeepFilesAccessedDays, "keep-files-access-days", 0, "purge files with access time greater than")
	rootCmd.PersistentFlags().StringVar(&ExternalRepoTargetList, "external-repo-target-list", "", "path to file containing list of external repo targets to keep")
	rootCmd.PersistentFlags().BoolVar(&Verbose, "verbose", false, "set verbosity for understanding what this script is doing")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "clean",
		Short: "clean bazel cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			log := createLogger(Verbose)
			log.Infof("Starting Clean up Process of Bazel Directory %s. This may take a couple of minutes", BazelCacheDir)

			activeExternalRepoTargets, err := generateExternalRepoTargets(ExternalRepoTargetList)
			if err != nil {
				return fmt.Errorf("failing to find external repo targets %v", err)
			}

			blackListDirectories := map[string]string{
				"install":        "", // install base for bazel
				"embedded_tools": "", // bazel dev tools
				"external":       "", // we can't purge this external repo. need to selectively purge
			}

			blackListFiles := map[string]string{
				"lock": "", // bazel lock
			}

			cleanBazelInput := CleanBazelInput{
				BazelCacheDir:             BazelCacheDir,
				KeepFilesAccessedDays:     KeepFilesAccessedDays,
				ActiveExternalRepoTargets: activeExternalRepoTargets,
				BlackListFiles:            blackListFiles,
				BlackListDirectories:      blackListDirectories,
			}

			filesToRemove, err := findFilesToClean(cleanBazelInput, log)
			if err != nil {
				return fmt.Errorf("failing to find files to clean: %v", err)
			}
			for _, file := range filesToRemove {
				err := os.RemoveAll(file)
				if err != nil {
					return fmt.Errorf("failing to remove file err: %v", err)
				}
			}
			log.Info("Finished Cleaning Bazel Cache up!")
			return nil
		},
	})
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
