// Package roles contains message roles.
package roles

const (
	System    = "system"
	Developer = "developer" // used in place of "system" for newer models
	User      = "user"
	Assistant = "assistant"
	AI        = Assistant // alias for Assistant
	Function  = "function"
	Tool      = "tool"
)
