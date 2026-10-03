package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

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

func TestStyleBar(t *testing.T) {
	edit := newTestEdit(t)
	edit.entry.SetText("Title\n\nSome text")
	edit.toggleRich()

	edit.lineStyle(widget.RichTextStyleHeading)()
	assert.Equal(t, "# Title\n\nSome text\n", edit.text())
	edit.clearStyle()
	assert.Equal(t, "Title\n\nSome text\n", edit.text())

	// with nothing selected the style is used for the text that is typed next
	edit.rich.SetText("")
	edit.toggleStyle(fyne.KeyB, 0)()
	test.Type(edit.rich, "bold")
	edit.toggleStyle(fyne.KeyB, 0)()
	test.Type(edit.rich, " text")
	assert.Equal(t, "**bold** text\n", edit.text())
}

func TestLineBlock(t *testing.T) {
	edit := newTestEdit(t)
	edit.entry.SetText("one\n\ntwo\n\nthree")
	edit.toggleRich()
	assert.Equal(t, "one\ntwo\nthree", edit.rich.Text)

	edit.rich.CursorRow = 1
	edit.rich.Refresh()
	edit.lineBlock(listItem(false))()
	assert.Equal(t, "one\n\n- two\n\nthree\n", edit.text())
	assert.Equal(t, "one\ntwo\nthree", edit.rich.Text)
	assert.Equal(t, 4, edit.rich.CursorTextOffset())

	// the next line joins the list above it
	edit.rich.CursorRow = 2
	edit.rich.CursorColumn = 2
	edit.rich.Refresh()
	edit.lineBlock(listItem(false))()
	assert.Equal(t, "one\n\n- two\n- three\n", edit.text())
	assert.Equal(t, 10, edit.rich.CursorTextOffset())

	// a line inside a list is left as it is
	edit.lineBlock(codeBlock)()
	assert.Equal(t, "one\n\n- two\n- three\n", edit.text())

	edit.rich.CursorRow = 0
	edit.rich.CursorColumn = 0
	edit.rich.Refresh()
	edit.lineBlock(codeBlock)()
	assert.Equal(t, "```\none\n```\n\n- two\n- three\n", edit.text())
	assert.Equal(t, "one\ntwo\nthree", edit.rich.Text)
}

func TestLineBlock_Styled(t *testing.T) {
	edit := newTestEdit(t)
	edit.entry.SetText("# Title\n\nSome **bold** text\n\nend")
	edit.toggleRich()

	edit.rich.CursorRow = 1
	edit.rich.CursorColumn = 7
	edit.rich.Refresh()
	edit.lineBlock(listItem(true))()
	assert.Equal(t, "# Title\n\n1. Some **bold** text\n\nend\n", edit.text())
	assert.Equal(t, 13, edit.rich.CursorTextOffset())

	// an empty last line can start a list too
	edit.rich.SetText("")
	edit.lineBlock(listItem(false))()
	assert.Equal(t, "- \n", edit.text())
}

// selectLines selects from the cursor over the given number of lines, up when negative.
func selectLines(edit *textEdit, lines int) {
	move := fyne.KeyDown
	if lines < 0 {
		move, lines = fyne.KeyUp, -lines
	}

	keys := make([]fyne.KeyName, lines)
	for i := range keys {
		keys[i] = move
	}
	selectWith(edit, keys...)
}

// selectWith presses the keys with shift held, extending the selection.
func selectWith(edit *textEdit, keys ...fyne.KeyName) {
	edit.rich.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	for _, key := range keys {
		edit.rich.TypedKey(&fyne.KeyEvent{Name: key})
	}
	edit.rich.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
}

func TestSelectedLines(t *testing.T) {
	edit := newTestEdit(t)
	edit.entry.SetText("one\n\ntwo\n\nthree\n\nfour")
	edit.toggleRich()

	start, end := edit.selectedLines()
	assert.Equal(t, 0, start)
	assert.Equal(t, 3, end)

	// selecting down from part way along a line
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	selectLines(edit, 2)
	assert.Equal(t, "ne\ntwo\nt", edit.rich.SelectedText())
	start, end = edit.selectedLines()
	assert.Equal(t, 0, start)
	assert.Equal(t, 13, end)

	// selecting up moves the cursor to the start of the selection
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	selectLines(edit, -1)
	assert.Equal(t, 7, edit.rich.CursorTextOffset())
	start, end = edit.selectedLines()
	assert.Equal(t, 4, start)
	assert.Equal(t, 13, end)

	// a selection ending at the cursor, with list markers left out of the count
	edit.lineBlock(listItem(false))()
	assert.Equal(t, "one\n\n- two\n- three\n\nfour\n", edit.text())
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	selectWith(edit, fyne.KeyDown, fyne.KeyDown, fyne.KeyDown, fyne.KeyEnd)
	assert.Equal(t, "\n\u2022 two\n\u2022 three\nfour", edit.rich.SelectedText())
	start, end = edit.selectedLines()
	assert.Equal(t, 0, start)
	assert.Equal(t, 18, end)
}

func TestLineBlock_Selection(t *testing.T) {
	edit := newTestEdit(t)
	edit.entry.SetText("one\n\ntwo\n\nthree\n\nfour")
	edit.toggleRich()

	selectLines(edit, 1)
	edit.lineStyle(widget.RichTextStyleSubHeading)()
	assert.Equal(t, "## one\n\n## two\n\nthree\n\nfour\n", edit.text())

	edit.rich.ClearSelection()
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	selectLines(edit, 1)
	edit.lineBlock(listItem(true))()
	assert.Equal(t, "## one\n\n## two\n\n1. three\n2. four\n", edit.text())
	assert.Equal(t, "one\ntwo\nthree\nfour", edit.rich.Text)

	edit.rich.SetText("a := 1\nb := 2\nc := 3")
	selectLines(edit, 2)
	edit.lineBlock(codeBlock)()
	assert.Equal(t, "```\na := 1\nb := 2\nc := 3\n```\n", edit.text())
}

func TestClearStyle_Blocks(t *testing.T) {
	edit := newTestEdit(t)
	edit.entry.SetText("1. **one**\n2. two\n3. three\n\n```\na := 1\nb := 2\nc := 3\n```")
	edit.toggleRich()
	assert.Equal(t, "1. **one**\n2. two\n3. three\n\n```\na := 1\nb := 2\nc := 3\n```\n", edit.text())

	// the middle item leaves the list, which starts again after it
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	edit.clearStyle()
	assert.Equal(t, "1. **one**\n\ntwo\n\n1. three\n\n```\na := 1\nb := 2\nc := 3\n```\n", edit.text())
	assert.Equal(t, "one\ntwo\nthree\na := 1\nb := 2\nc := 3", edit.rich.Text)
	assert.Equal(t, 4, edit.rich.CursorTextOffset())

	// the first item loses its style as well as its number
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	edit.clearStyle()
	assert.Equal(t, "one\n\ntwo\n\n1. three\n\n```\na := 1\nb := 2\nc := 3\n```\n", edit.text())

	// the middle line of the code block splits it in two
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	edit.clearStyle()
	assert.Equal(t, "one\n\ntwo\n\n1. three\n\n```\na := 1\n```\n\nb := 2\n\n```\nc := 3\n```\n", edit.text())

	// a selection clears every line it touches
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	selectLines(edit, 2)
	edit.clearStyle()
	assert.Equal(t, "one\n\ntwo\n\nthree\n\na := 1\n\nb := 2\n\n```\nc := 3\n```\n", edit.text())
}

func TestClearStyle_Selection(t *testing.T) {
	edit := newTestEdit(t)
	edit.entry.SetText("- Some **bold** and *italic* text")
	edit.toggleRich()

	// only the selected text is cleared, the rest of the line is left alone
	for i := 0; i < 5; i++ {
		edit.rich.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	}
	selectWith(edit, fyne.KeyRight, fyne.KeyRight, fyne.KeyRight, fyne.KeyRight)
	assert.Equal(t, "bold", edit.rich.SelectedText())
	edit.clearStyle()
	assert.Equal(t, "- Some bold and *italic* text\n", edit.text())

	// without a selection the whole line is cleared
	edit.rich.ClearSelection()
	edit.clearStyle()
	assert.Equal(t, "Some bold and italic text\n", edit.text())
}
