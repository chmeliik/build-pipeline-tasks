package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/yaml"
)

func main() {
	var backupDir string
	var formatBackup bool

	flag.StringVar(&backupDir, "backup-dir", "", "directory in which to backup the target files")
	flag.BoolVar(&formatBackup, "format-backup", false, "format the backup files")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [PIPELINES_DIR]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  PIPELINES_DIR\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    \tpath to root pipelines directory, defaults to ../pipelines\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	pipelinesDir := flag.Arg(0)
	if pipelinesDir == "" {
		pipelinesDir = "../pipelines"
	}

	p := func(path string) string {
		return filepath.Join(pipelinesDir, path)
	}

	dockerBuild := unwrap(readPipeline(p("docker-build/docker-build.yaml")))

	dockerBuildOciTa := unwrap(generate(genArgs{
		fn:           GenerateDockerBuildOciTa,
		source:       dockerBuild,
		destPath:     p("docker-build-oci-ta/docker-build-oci-ta.yaml"),
		backupDir:    backupDir,
		formatBackup: formatBackup,
	}))

	unwrap(generate(genArgs{
		fn:           GenerateDockerBuildMultiPlatform,
		source:       dockerBuildOciTa,
		destPath:     p("docker-build-multi-platform-oci-ta/docker-build-multi-platform-oci-ta.yaml"),
		backupDir:    backupDir,
		formatBackup: formatBackup,
	}))

	unwrap(generate(genArgs{
		fn:           GenerateDockerBuildMin,
		source:       dockerBuildOciTa,
		destPath:     p("docker-build-oci-ta-min/docker-build-oci-ta-min.yaml"),
		backupDir:    backupDir,
		formatBackup: formatBackup,
	}))
}

func unwrap[T any](v T, e error) T {
	if e != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", e)
		os.Exit(1)
	}
	return v
}

type genArgs struct {
	fn           func(source tektonapi.Pipeline, dest *tektonapi.Pipeline) (tektonapi.Pipeline, error)
	source       tektonapi.Pipeline
	destPath     string
	backupDir    string
	formatBackup bool
}

func generate(args genArgs) (tektonapi.Pipeline, error) {
	var result tektonapi.Pipeline

	log.Println("generating", args.destPath)

	if destPipeline, err := readPipeline(args.destPath); err == nil {
		result, err = args.fn(args.source, &destPipeline)
		if err != nil {
			return tektonapi.Pipeline{}, fmt.Errorf("re-generating existing %s: %w", args.destPath, err)
		}
		if args.backupDir != "" {
			if err := backup(destPipeline, args.destPath, args.backupDir, args.formatBackup); err != nil {
				return tektonapi.Pipeline{}, fmt.Errorf("backing up %s: %w", args.destPath, err)
			}
		}
	} else if errors.Is(err, os.ErrNotExist) {
		result, err = args.fn(args.source, nil)
		if err != nil {
			return tektonapi.Pipeline{}, fmt.Errorf("generating new %s: %w", args.destPath, err)
		}
	} else {
		return tektonapi.Pipeline{}, err
	}

	if err := writePipeline(result, args.destPath); err != nil {
		return tektonapi.Pipeline{}, fmt.Errorf("writing %s: %w", args.destPath, err)
	}

	return result, nil
}

func readPipeline(path string) (tektonapi.Pipeline, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return tektonapi.Pipeline{}, err
	}

	var pipeline tektonapi.Pipeline
	err = yaml.Unmarshal(bytes, &pipeline)
	if err != nil {
		return tektonapi.Pipeline{}, fmt.Errorf("unmarshaling %s: %w", path, err)
	}

	return pipeline, nil
}

const warningHeader = "# WARNING: This is an auto generated file, do not modify this file directly"

func writePipeline(pipeline tektonapi.Pipeline, path string) error {
	bytes, err := yaml.Marshal(pipeline)
	if err != nil {
		return err
	}
	content := append([]byte(warningHeader+"\n"), bytes...)
	return os.WriteFile(path, content, 0644)
}

func backup(pipeline tektonapi.Pipeline, destPath string, backupDir string, format bool) error {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("creating backup dir: %w", err)
	}

	backupPath := filepath.Join(backupDir, filepath.Base(destPath))

	if format {
		return writePipeline(pipeline, backupPath)
	} else {
		bytes, err := os.ReadFile(destPath)
		if err != nil {
			return err
		}
		return os.WriteFile(backupPath, bytes, 0644)
	}
}
