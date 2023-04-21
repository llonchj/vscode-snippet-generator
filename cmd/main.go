package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var SpacesIndent string
var OutputDir string

const VSCodeSnippetsFolder = "Code/User/snippets"

func GetDefaultOutputDirectory() string {
	/*
		See https://code.visualstudio.com/docs/getstarted/settings#_settings-file-locations
	*/
	home, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, VSCodeSnippetsFolder)
}

func init() {
	const spacesIndent = "    "

	flag.StringVar(&SpacesIndent, "i", spacesIndent, "indentation")
	flag.StringVar(&OutputDir, "o", GetDefaultOutputDirectory(), "path to VS Code snippets folder.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage:\n  %s [flags] (FILE|DIR)...\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}
}

type Body []byte

func (b *Body) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.Split(strings.TrimRight(string(*b), "\n"), "\n"))
}

type File struct {
	Prefix      string `json:"prefix"`
	Description string `json:"description"`
	Body        Body   `json:"body"`
}

type Snippet map[string]*File

func (s *Snippet) AddFile(pathName string) error {
	_, fileName := filepath.Split(pathName)
	baseName := fileName[:len(fileName)-len(filepath.Ext(fileName))]

	b, err := os.ReadFile(pathName)
	if err != nil {
		return fmt.Errorf("reading %s: %w", pathName, err)
	}

	(*s)[baseName] = &File{
		Prefix:      baseName,
		Description: "",
		Body:        Body(b),
	}
	return nil
}

type Snippets map[string]*Snippet

func (s *Snippets) AddSnippet(pathName string) error {
	ext := filepath.Ext(pathName)[1:]
	_, ok := (*s)[ext]
	if !ok {
		(*s)[ext] = &Snippet{}
	}
	return ((*s)[ext]).AddFile(pathName)
}

func (s *Snippets) Write(pathName string) error {
	for k, v := range *s {
		fileName := filepath.Join(pathName, k+".json")
		f, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("creating %s: %w", fileName, err)
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", SpacesIndent)
		if err := enc.Encode(v); err != nil {
			return fmt.Errorf("encoding %s: %w", fileName, err)
		}
	}
	return nil
}

func process(ctx context.Context) error {
	snippets := Snippets{}
	for _, pathName := range os.Args[1:] {
		if err := filepath.Walk(pathName, func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			return snippets.AddSnippet(path)
		}); err != nil {
			return fmt.Errorf("walking %s: %w", pathName, err)
		}
	}

	// create output folder if does not exist.
	if _, err := os.Stat(OutputDir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(OutputDir, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", OutputDir, err)
		}
	}

	return snippets.Write(OutputDir)
}

func main() {
	flag.Parse()

	ctx := context.Background()
	if err := process(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
