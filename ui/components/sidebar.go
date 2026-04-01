package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// Sidebar renders the left-hand navigation panel.
type Sidebar struct {
	app.Compo
	Links []NavLink
}

func (s *Sidebar) Render() app.UI {
	items := app.Range(s.Links).Slice(func(i int) app.UI {
		return app.Li().Body(
			app.A().Href(s.Links[i].Href).Text(s.Links[i].Text),
		)
	})

	return app.Aside().Body(
		app.P().Class("sidebar-label").Text("Navigation"),
		app.Nav().Body(
			app.Ul().Body(items),
		),
	)
}
