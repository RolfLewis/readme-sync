package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/rolflewis/readme-sync/docs"
	"github.com/rolflewis/readme-sync/readme"
	"golang.org/x/xerrors"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("No arguments provided")
		return
	}

	if err := godotenv.Load(); err != nil {
		fmt.Printf("%+v", err)
		return
	}

	command := os.Args[1]
	arguments := os.Args[2:]

	definition, valid := commands[command]
	if !valid {
		fmt.Printf("Invalid command provided: %v\n", command)
		return
	}

	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	if err := definition.function(context.Background(), flags, arguments); err != nil {
		fmt.Printf("%+v", err)
	}
}

func walk(ctx context.Context, fs *flag.FlagSet, args []string) error {
	var path string
	fs.StringVar(&path, "path", "", "path to docs root")

	if err := fs.Parse(args); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	if path == "" {
		return xerrors.New("empty path")
	}

	client, err := readme.NewClient(ctx, os.Getenv("README_APIKEY"), "")
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	catalog, err := docs.WalkCatalog(ctx, path)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	for cat := range catalog.Categories {
		if err := docs.ProcessCategory(ctx, client, cat); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	}

	for _, doc := range catalog.Docs {
		if doc.Parent == "" {
			log.Println("processing")
			if err := docs.ProcessDoc(ctx, client, doc); err != nil {
				return xerrors.Errorf(": %w", err)
			}
		}
	}

	for _, doc := range catalog.Docs {
		if doc.Parent != "" {
			log.Println("processing child")
			if err := docs.ProcessDoc(ctx, client, doc); err != nil {
				return xerrors.Errorf(": %w", err)
			}
		}
	}

	return nil
}

type commandDescription struct {
	title   string
	body    string
	example string
}

type commandFunction func(ctx context.Context, fs *flag.FlagSet, args []string) error

type commandDefinition struct {
	desc     commandDescription
	function commandFunction
}

var commands = map[string]commandDefinition{
	"walk": {
		desc: commandDescription{
			"Walk local docs folder, testing",
			"this is filler",
			"readme-sync walk ./path/to/docs/root",
		},
		function: walk,
	},
}
