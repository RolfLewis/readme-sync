package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rolflewis/readme-sync/config"
	"github.com/rolflewis/readme-sync/page"
	"golang.org/x/xerrors"
)

const defaultConfigFile = "config.yml"
const defaultSourceDirectory = "docs"
const defaultApiKey = "apikey"
const defaultTargetVersion = "version"

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

	log.Println(command, arguments)
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

func push(ctx context.Context, fs *flag.FlagSet, args []string) error {
	var prune, dry, force, unhide bool
	var configPath, sourcePath, categoryList string

	fs.BoolVar(&prune, "prune", false, "if set, remotes pages with no local counterpart will be removed")
	fs.BoolVar(&dry, "dry", false, "if set, performs all processing short of modifying remote resources")
	fs.BoolVar(&force, "force", false, "if set, skips change detection and pushes all pages")
	fs.BoolVar(&unhide, "unhide", false, "if set, overrides hiding on pages and exposes all")
	fs.StringVar(&configPath, "config", "", "path to configuration file")
	fs.StringVar(&sourcePath, "source", "docs", "path to source file/directory")
	fs.StringVar(&categoryList, "categories", "", "comma separated list of categories")

	if err := fs.Parse(args); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	fmt.Println(prune, dry, force, unhide, sourcePath, categoryList)

	categories := strings.Split(categoryList, ",")

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	// only use global categories list if nothing was passed to command
	if len(categories) == 0 {
		categories = cfg.Categories
	}

	// no point in running if we have no categories
	if len(categories) == 0 {
		return xerrors.New("no categories provided")
	}

	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	if !fileInfo.IsDir() {
		if err := page.ProcessPage(ctx, sourcePath); err != nil {
			return xerrors.Errorf(": %w", err)
		}
		return nil
	}

	// unimplemented
	return xerrors.New("not implemented")

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
	"push": {
		desc: commandDescription{
			"Push Documents to Remote ReadMe Site",
			"This is filler",
			"readme-sync push [--prune]",
		},
		function: push,
	},
}
