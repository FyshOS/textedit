package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"fyne.io/x/fyne/menu"
)

func newTestEdit(t *testing.T) *textEdit {
	test.NewTempApp(t)
	w := test.NewTempWindow(t, nil)
	edit := &textEdit{window: w, changed: binding.NewBool()}
	edit.recents = menu.NewRecents("Recent items...", nil)
	ui := edit.makeUI(w)
	w.SetContent(ui)
	return edit
}

func TestLoad(t *testing.T) {
	edit := newTestEdit(t)

	r, err := storage.Reader(storage.NewFileURI("./testdata/test.txt"))
	assert.Nil(t, err)
	err = edit.load(r)
	assert.Nil(t, err)

	assert.Equal(t, "Test content", edit.entry.Text)
}

func TestSave(t *testing.T) {
	edit := newTestEdit(t)

	out, err := storage.Writer(storage.NewFileURI("./testdata/test2.txt"))
	assert.Nil(t, err)
	defer os.Remove("./testdata/test2.txt")

	edit.entry.SetText("Testing")
	err = edit.saveAs(out)
	assert.Nil(t, err)
	out.Close()

	data, err := os.ReadFile("./testdata/test2.txt")
	assert.Nil(t, err)
	assert.Equal(t, "Testing", string(data))

	edit.entry.SetText("Testing2")
	edit.save()
	data, err = os.ReadFile("./testdata/test2.txt")
	assert.Nil(t, err)
	assert.Equal(t, "Testing2", string(data))
}

func TestLoad_Markdown(t *testing.T) {
	edit := newTestEdit(t)

	r, err := storage.Reader(storage.NewFileURI("./testdata/test.md"))
	assert.Nil(t, err)
	err = edit.load(r)
	assert.Nil(t, err)

	assert.True(t, edit.richMode)
	assert.True(t, edit.richItem.Checked)
	assert.Equal(t, "Title\nSome bold text", edit.rich.Text)
	assert.Equal(t, "# Title\n\nSome **bold** text\n", edit.text())

	r, err = storage.Reader(storage.NewFileURI("./testdata/test.txt"))
	assert.Nil(t, err)
	err = edit.load(r)
	assert.Nil(t, err)

	assert.False(t, edit.richMode)
	assert.False(t, edit.richItem.Checked)
	assert.Equal(t, "Test content", edit.entry.Text)
}

func TestSave_Markdown(t *testing.T) {
	edit := newTestEdit(t)

	edit.entry.SetText("Some *styled* text")
	edit.toggleRich()
	assert.True(t, edit.richMode)
	assert.Equal(t, "Some styled text", edit.rich.Text)

	out, err := storage.Writer(storage.NewFileURI("./testdata/test2.md"))
	assert.Nil(t, err)
	defer os.Remove("./testdata/test2.md")

	err = edit.saveAs(out)
	assert.Nil(t, err)

	data, err := os.ReadFile("./testdata/test2.md")
	assert.Nil(t, err)
	assert.Equal(t, "Some *styled* text\n", string(data))
}

func TestToggleRich(t *testing.T) {
	edit := newTestEdit(t)

	assert.False(t, edit.richMode)
	assert.Equal(t, "Untitled.txt", edit.fileName())

	edit.entry.SetText("# Heading")
	edit.toggleRich()
	assert.True(t, edit.richMode)
	assert.Equal(t, "Heading", edit.rich.Text)
	assert.Equal(t, "Untitled.md", edit.fileName())

	// the styles are kept as markdown when going back to plain text
	edit.toggleRich()
	assert.False(t, edit.richMode)
	assert.Equal(t, "# Heading", edit.entry.Text)
	assert.Equal(t, "Untitled.txt", edit.fileName())
}

func TestToggleRich_Rename(t *testing.T) {
	edit := newTestEdit(t)

	r, err := storage.Reader(storage.NewFileURI("./testdata/test.txt"))
	assert.Nil(t, err)
	err = edit.load(r)
	assert.Nil(t, err)
	assert.False(t, edit.rename)

	edit.toggleRich()
	assert.True(t, edit.rename)
	assert.Equal(t, "test.md", edit.fileName())

	edit.toggleRich()
	assert.False(t, edit.rename)

	// a markdown file keeps its name when the source is edited as plain text
	r, err = storage.Reader(storage.NewFileURI("./testdata/test.md"))
	assert.Nil(t, err)
	err = edit.load(r)
	assert.Nil(t, err)

	edit.toggleRich()
	assert.False(t, edit.richMode)
	assert.False(t, edit.rename)
	assert.Equal(t, "# Title\n\nSome **bold** text", edit.entry.Text)
}
