package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// PageLayout provides the standard page structure: Header → Main > grid(Sidebar, Article).
// Pass service-specific Header and Sidebar as app.UI values.
type PageLayout struct {
	app.Compo
	Header  app.UI
	Sidebar app.UI
	Content app.UI
}

func (p *PageLayout) Render() app.UI {
	return app.Div().Body(
		p.Header,
		app.Main().Class("container").Body(
			app.Div().Class("grid").Body(
				p.Sidebar,
				app.Article().Body(p.Content),
			),
		),
	)
}
