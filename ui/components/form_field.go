package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// FormField renders a labeled form field wrapping any input element.
type FormField struct {
	app.Compo
	Label    string
	ID       string
	Required bool
	Input    app.UI
}

func (f *FormField) Render() app.UI {
	return app.Div().Class("form-group").Body(
		app.Label().For(f.ID).Text(f.Label),
		f.Input,
	)
}
