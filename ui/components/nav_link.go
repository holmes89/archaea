// Package components provides reusable go-app UI components shared across services.
package components

// NavLink is a single navigation entry used by Header and Sidebar.
type NavLink struct {
	Href string
	Text string
}
