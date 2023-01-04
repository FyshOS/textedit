package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
)

func TestLoad(t *testing.T) {
	w := test.NewWindow(nil)
	edit := &textEdit{window: w, changed: binding.NewBool()}
	ui := edit.makeUI()
	w.SetContent(ui)

	r, err := storage.Reader(storage.NewFileURI("./testdata/test.txt"))
	assert.Nil(t, err)
	err = edit.load(r)
	assert.Nil(t, err)

	assert.Equal(t, "Test content", edit.entry.Text)
}

func TestSave(t *testing.T) {
	w := test.NewWindow(nil)
	edit := &textEdit{window: w, changed: binding.NewBool()}
	ui := edit.makeUI()
	w.SetContent(ui)

	out, err := storage.Writer(storage.NewFileURI("./testdata/test2.txt"))
	assert.Nil(t, err)
	defer os.Remove("./testdata/test2.txt")

	edit.entry.SetText("Testing")
	err = edit.saveAs(out)
	assert.Nil(t, err)
	out.Close()

	data, err := ioutil.ReadFile("./testdata/test2.txt")
	assert.Nil(t, err)
	assert.Equal(t, "Testing", string(data))

	edit.entry.SetText("Testing2")
	edit.save()
	data, err = ioutil.ReadFile("./testdata/test2.txt")
	assert.Nil(t, err)
	assert.Equal(t, "Testing2", string(data))
}
