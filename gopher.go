/*
 * Copyright © 2020 A Bunch Tell LLC.
 *
 * This file is part of WriteFreely.
 *
 * WriteFreely is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, included
 * in the LICENSE file in this source code package.
 */

package writefreely

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/prologic/go-gopher"
	"github.com/writeas/web-core/log"
)

func initGopher(apper Apper) {
	handler := NewWFHandler(apper)

	gopher.HandleFunc("/", handler.Gopher(handleGopher))
	log.Info("Serving on gopher://localhost:%d", apper.App().Config().Server.GopherPort)
	gopher.ListenAndServe(fmt.Sprintf(":%d", apper.App().Config().Server.GopherPort), nil)
}

func handleGopher(app *App, w gopher.ResponseWriter, r *gopher.Request) error {
	parts := strings.Split(r.Selector, "/")
	if app.cfg.App.SingleUser {
		if parts[1] != "" {
			return handleGopherCollectionPost(app, w, r)
		}
		return handleGopherCollection(app, w, r)
	}

	// Show all public collections (a gopher Reader view, essentially)
	if len(parts) == 3 {
		return handleGopherCollection(app, w, r)
	}

	w.WriteInfo(fmt.Sprintf("Welcome to %s", app.cfg.App.SiteName))

	colls, err := app.db.GetPublicCollections(app.cfg.App.Host)
	if err != nil {
		return err
	}

	for _, c := range *colls {
		w.WriteItem(&gopher.Item{
			Type:        gopher.DIRECTORY,
			Description: c.DisplayTitle(),
			Selector:    "/" + c.Alias + "/",
		})
	}
	return w.End()
}

func handleGopherCollection(app *App, w gopher.ResponseWriter, r *gopher.Request) error {
	var (
		c               *Collection
		collAlias, slug string
		baseSel         = "/"
		err             error
	)

	parts := strings.Split(r.Selector, "/")
	if app.cfg.App.SingleUser {
		// sanity check
		slug = parts[1]
		if slug != "" {
			return handleGopherCollectionPost(app, w, r)
		}

		c, err = app.db.GetCollectionByID(1)
		if err != nil {
			return err
		}
	} else {
		collAlias = parts[1]
		slug = parts[2]
		if slug != "" {
			return handleGopherCollectionPost(app, w, r)
		}

		c, err = app.db.GetCollection(collAlias)
		if err != nil {
			return err
		}
		baseSel = "/" + c.Alias + "/"
	}
	c.hostName = app.cfg.App.Host

	posts, err := app.db.GetPosts(app.cfg, c, 0, false, false, false)
	if err != nil {
		return err
	}

	for _, p := range *posts {
		w.WriteItem(&gopher.Item{
			Type:        gopher.FILE,
			Description: p.CreatedDate() + " - " + p.DisplayTitle(),
			Selector:    baseSel + p.Slug.String,
		})
	}
	return w.End()
}

func handleGopherCollectionPost(app *App, w gopher.ResponseWriter, r *gopher.Request) error {
	var (
		c               *Collection
		collAlias, slug string
		err             error
	)

	parts := strings.Split(r.Selector, "/")
	if app.cfg.App.SingleUser {
		slug = parts[1]
		c, err = app.db.GetCollectionByID(1)
		if err != nil {
			return err
		}
	} else {
		collAlias = parts[1]
		slug = parts[2]
		c, err = app.db.GetCollection(collAlias)
		if err != nil {
			return err
		}
	}
	c.hostName = app.cfg.App.Host

	p, err := app.db.GetPost(slug, c.ID)
	if err != nil {
		return err
	}

	b := bytes.Buffer{}
	if p.Title.String != "" {
		b.WriteString(p.Title.String + "\n")
	}
	b.WriteString(p.DisplayDate + "\n\n")
	b.WriteString(p.Content)
	io.Copy(w, &b)

	return w.End()
}
