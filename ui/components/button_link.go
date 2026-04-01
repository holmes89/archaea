package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// ButtonLink renders an anchor styled as a button.
// Variant may be "primary" (default), "outline", "contrast", or "contrast outline".
type ButtonLink struct {
	app.Compo
	Href    string
	Text    string
	Variant string
}

func (b *ButtonLink) Render() app.UI {
	variant := b.Variant
	if variant == "" {
		variant = "primary"
	}
	link := app.A().Href(b.Href).Role("button").Text(b.Text)
	if variant != "primary" {
		link = link.Class(variant)
	}
	return link
}
