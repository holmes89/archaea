package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// LoadingState renders one of three states: a loading spinner, an error
// message, or the provided content.
type LoadingState struct {
	app.Compo
	Loading bool
	Error   string
	Content app.UI
}

func (l *LoadingState) Render() app.UI {
	return app.Div().Body(
		app.If(l.Loading, func() app.UI {
			return app.Div().Class("loading").Text("Loading...")
		}),
		app.If(l.Error != "", func() app.UI {
			return app.Div().Class("error").Text(l.Error)
		}),
		app.If(!l.Loading && l.Error == "", func() app.UI {
			return l.Content
		}),
	)
}
