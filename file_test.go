package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
)

func TestLoad(t *testing.T) {
	w := test.NewWindow(nil)
	edit, ui := makeUI(w)
	w.SetContent(ui)

	r, err := storage.Reader(storage.NewFileURI("./testdata/test.txt"))
	assert.Nil(t, err)
	err = edit.load(r)
	assert.Nil(t, err)

	assert.Equal(t, "Test content", edit.entry.Text)
}