package action

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-registry/alpha/declcfg"

	"github.com/operator-framework/kubectl-operator/internal/pkg/catalogd/fetcher"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// CatalogList is a helper struct that implements
// functionality to fetch catalog contents for the
// "catalog list" subcommand
type CatalogList struct {
	config             *action.Configuration
	Logf               func(string, ...interface{})
	createFetcherFunc  func(*action.Configuration) CatalogFetcher
	createStreamerFunc func(*action.Configuration) CatalogContentStreamer
}

// CatalogListOptions is the set of configurable
// options that are used to filter the set of catalog
// contents to return
type CatalogListOptions struct {
	// Catalog is the name of the catalog
	// all returned contents should be from.
	// Optional.
	Catalog string
	// Schema is the schema that all the returned
	// contents should have.
	// Optional.
	Schema string
	// Package is the package that all the returned
	// contents should be from.
	// Optional.
	Package string
	// Name is the name that all returned
	// contents should have.
	// Optional.
	Name string
}

func NewCatalogList(cfg *action.Configuration, fetcherFunc FetcherFunc, streamerFunc StreamerFunc) *CatalogList {
	return &CatalogList{
		config:             cfg,
		createFetcherFunc:  fetcherFunc,
		createStreamerFunc: streamerFunc,
		Logf:               func(string, ...interface{}) {},
	}
}

// Run will return a list of catalog objects using the Meta type that match the provided options.
// Returns nil and an error if any are encountered.
func (i *CatalogList) Run(ctx context.Context, listOpts CatalogListOptions) ([]Meta, error) {
	fetch := i.createFetcherFunc(i.config)
	stream := i.createStreamerFunc(i.config)

	catalogs, err := fetch.FetchCatalogs(ctx,
		fetcher.WithNameFilter(listOpts.Catalog),
		fetcher.WithUnpackedFilter(),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching catalogs: %w", err)
	}

	metas := []Meta{}

	for _, catalog := range catalogs {
		catalogName := catalog.Name
		rc, err := stream.StreamCatalogContents(ctx, catalog)
		if err != nil {
			return nil, fmt.Errorf("streaming FBC for catalog %q: %w", catalog.Name, err)
		}
		err = declcfg.WalkMetasReader(rc, func(meta *declcfg.Meta, err error) error {
			if err != nil {
				return err
			}

			if listOpts.Schema != "" && meta.Schema != listOpts.Schema {
				return nil
			}

			if listOpts.Package != "" && meta.Package != listOpts.Package {
				return nil
			}

			if listOpts.Name != "" && meta.Name != listOpts.Name {
				return nil
			}

			metaEntry := Meta{
				Meta: declcfg.Meta{
					Schema:  meta.Schema,
					Package: meta.Package,
					Name:    meta.Name,
					Blob:    meta.Blob,
				},
				Catalog: catalogName,
			}
			metas = append(metas, metaEntry)

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("reading FBC for catalog %q: %w", catalog.Name, err)
		}
		rc.Close()
	}

	return metas, nil
}
