package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type textEdit struct {
	cursorRow, cursorCol *widget.Label
	entry                *widget.Entry
	window               fyne.Window
	changed              binding.Bool

	uri fyne.URI
}

func (e *textEdit) updateStatus() {
	e.cursorRow.SetText(fmt.Sprintf("%d", e.entry.CursorRow+1))
	e.cursorCol.SetText(fmt.Sprintf("%d", e.entry.CursorColumn+1))
}

func (e *textEdit) cut() {
	e.entry.TypedShortcut(&fyne.ShortcutCut{Clipboard: fyne.CurrentApp().Clipboard()})
}

func (e *textEdit) copy() {
	e.entry.TypedShortcut(&fyne.ShortcutCopy{Clipboard: fyne.CurrentApp().Clipboard()})
}

func (e *textEdit) paste() {
	e.entry.TypedShortcut(&fyne.ShortcutPaste{Clipboard: fyne.CurrentApp().Clipboard()})
}

func (e *textEdit) buildToolbar() *widget.Toolbar {
	return widget.NewToolbar(
		widget.NewToolbarAction(theme.FolderOpenIcon(), e.open),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), e.save),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			e.entry.SetText("")
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ContentCutIcon(), e.cut),
		widget.NewToolbarAction(theme.ContentCopyIcon(), e.copy),
		widget.NewToolbarAction(theme.ContentPasteIcon(), e.paste),
	)
}

// makeUI loads a new text editor
func (e *textEdit) makeUI() fyne.CanvasObject {
	e.entry = widget.NewMultiLineEntry()
	e.cursorRow = widget.NewLabel("1")
	e.cursorCol = widget.NewLabel("1")

	e.entry.OnCursorChanged = e.updateStatus
	e.entry.OnChanged = func(s string) {
		e.changed.Set(true)
	}

	toolbar := e.buildToolbar()
	status := container.NewHBox(layout.NewSpacer(),
		widget.NewLabel("Cursor Row:"), e.cursorRow,
		widget.NewLabel("Col:"), e.cursorCol)
	return container.NewBorder(toolbar, status, nil, nil, container.NewScroll(e.entry))
}
