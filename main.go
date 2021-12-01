//go:generate fyne bundle -o data.go img/Icon.png

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.New()
	a.SetIcon(resourceIconPng)
	w := a.NewWindow("TextEdit")

	w.SetContent(makeUI(w))
	w.Resize(fyne.NewSize(480, 360))
	w.ShowAndRun()
}
