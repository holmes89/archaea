// Package ui provides shared go-app bootstrap helpers for service UIs.
package ui

import (
	"log"
	"net/http"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Run configures the go-app HTTP handler and starts the server on addr.
// Call after all app.Route / app.RouteWithRegexp registrations.
//
// styles are CSS paths served under /web/css/, e.g.
// "/web/css/pico.amber.min.css", "/web/css/app.css".
//
// Typical usage:
//
//	app.Route("/", func() app.Composer { return &pages.HomePage{} })
//	app.RunWhenOnBrowser()
//	ui.Run("MyApp", "My app description", ":8000",
//	    "/web/css/pico.amber.min.css", "/web/css/app.css")
func Run(name, description, addr string, styles ...string) {
	http.Handle("/", &app.Handler{
		Name:        name,
		Description: description,
		Styles:      styles,
		RawHeaders: []string{
			`<script>document.documentElement.setAttribute('data-theme','light')</script>`,
		},
	})

	log.Printf("ui: starting %s on %s\n", name, addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
