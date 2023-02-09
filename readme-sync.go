package main

import (
	"context"
	"flag"
	"fmt"
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

	client, err := readme.NewClient(ctx, os.Getenv("README_APIKEY"), "")
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	catalog, err := docs.WalkCatalog(ctx, path)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	renameMap := make(map[string]string)
	renameMap["engineering"] = "Rename Test 1"

	for cat := range catalog.Categories {
		metadata := docs.CatMetadata{
			Slug:  cat,
			Title: cat,
		}

		title, ok := renameMap[cat]
		if ok {
			metadata.Title = title
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
			fmt.Printf("Pruning Category with Slug %v\n", cat.Slug)
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
				fmt.Printf("Pruning Doc with Slug %v\n", doc.Slug)
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
