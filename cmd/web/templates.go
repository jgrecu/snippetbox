package main

import (
	"snippetbox.jgrecu.eu/internal/models"
)

type templateData struct {
	Snippet  models.Snippet
	Snippets []models.Snippet
}
