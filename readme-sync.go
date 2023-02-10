package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/rolflewis/readme-sync/config"
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

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	if err := walk(context.Background(), flags, os.Args[1:]); err != nil {
		fmt.Printf("%+v", err)
		return
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

	cfg, err := config.NewConfig("")
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	client, err := readme.NewClient(ctx, cfg.Key, cfg.Version)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	catalog, err := docs.WalkCatalog(ctx, path)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	// Create the category config map
	catConfigs := make(map[string]config.CategoryConfig)
	for _, catCfg := range cfg.Categories {
		catConfigs[catCfg.Slug] = catCfg
	}

	// Make sure all categories in the catalog are represented in the config
	for cat := range catalog.Categories {
		if _, found := catConfigs[cat]; !found {
			msg := fmt.Sprintf("Top-level folder with slug \"%v\" does not have a matching category entry in the configuration file", cat)
			return xerrors.New(msg)
		}
	}

	// Make sure all categories in the config are represented in the catalog
	for cat := range catConfigs {
		if _, found := catalog.Categories[cat]; !found {
			msg := fmt.Sprintf("Category configuration with slug \"%v\" does not have a matching top-level folder in the provided path", cat)
			return xerrors.New(msg)
		}
	}

	for cat := range catalog.Categories {
		metadata := docs.CatMetadata{
			Slug:  cat,
			Title: cat,
		}

		catCfg, ok := catConfigs[cat]
		if ok && catCfg.Title != "" { // readme does not accept empty titles
			metadata.Title = catCfg.Title
		}

		if err := docs.ProcessCategory(ctx, client, metadata); err != nil {
			return xerrors.Errorf(": %w", err)
		}
	}

	for _, doc := range catalog.Docs {
		if doc.Parent == "" {
			if err := docs.ProcessDoc(ctx, client, doc); err != nil {
				return xerrors.Errorf(": %w", err)
			}
		}
	}

	for _, doc := range catalog.Docs {
		if doc.Parent != "" {
			if err := docs.ProcessDoc(ctx, client, doc); err != nil {
				return xerrors.Errorf(": %w", err)
			}
		}
	}

	if err := prune(context.Background(), client, catalog); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

func prune(ctx context.Context, client *readme.Client, catalog docs.Catalog) error {
	cats, err := client.GetCategories(ctx)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	for _, cat := range cats {
		// Deleting a category automatically removes all contained docs, saving time on the next step
		if _, found := catalog.Categories[cat.Slug]; !found {
			fmt.Printf("Pruning category with slug \"%v\"\n", cat.Slug)
			if err := client.DeleteCategory(ctx, cat.Slug); err != nil {
				return xerrors.Errorf(": %w", err)
			}
			continue
		}
		// Prune docs inside present categories, children first
		docs, err := client.GetDocsForCategory(ctx, cat.Slug)
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}

		pruneDoc := func(doc readme.Document) error {
			if _, found := catalog.Docs[doc.Slug]; !found {
				fmt.Printf("Pruning doc with slug \"%v\"\n", doc.Slug)
				if err := client.DeleteDoc(ctx, doc.Slug); err != nil {
					return xerrors.Errorf(": %w", err)
				}
			}
			return nil
		}
		// Children first
		for _, doc := range docs {
			if doc.Parent != "" {
				if err := pruneDoc(doc); err != nil {
					return xerrors.Errorf(": %w", err)
				}
			}
		}
		// Non-children last
		for _, doc := range docs {
			if doc.Parent == "" {
				if err := pruneDoc(doc); err != nil {
					return xerrors.Errorf(": %w", err)
				}
			}
		}
	}

	return nil
}
