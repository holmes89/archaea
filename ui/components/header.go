package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// Header renders the top navigation bar. Title is the brand name shown on the
// left; Links are rendered as nav items on the right.
type Header struct {
	app.Compo
	Title string
	Links []NavLink
}

func (h *Header) Render() app.UI {
	rightNav := app.Range(h.Links).Slice(func(i int) app.UI {
		return app.Li().Body(
			app.A().Href(h.Links[i].Href).Class("nav-link").Text(h.Links[i].Text),
		)
	})

	return app.Header().
		Class("container").
		Body(
			app.Nav().Body(
				app.Ul().Body(
					app.Li().Body(
						app.A().Href("/").Class("site-title").Body(
							app.Strong().Text(h.Title),
						),
					),
				),
				app.Ul().Body(rightNav),
			),
		)
}
