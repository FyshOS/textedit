package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type textEdit struct {
	cursorRow, cursorCol *widget.Label
	entry                *editorEntry
	rich                 *richEditorEntry
	content              *fyne.Container
	window               fyne.Window
	changed              binding.Bool

	// richMode is set when the document is styled text that is saved as markdown
	richMode   bool
	richItem   *fyne.MenuItem
	richToggle *toolbarToggle
	formatMenu *fyne.Menu

	uri fyne.URI
	// rename is set when a file that is not markdown was switched to rich mode,
	// so the next save asks for a name with the markdown extension
	rename bool
}

// cursor returns the entry that is currently being edited, for the state that both modes share.
func (e *textEdit) cursor() *widget.Entry {
	if e.richMode {
		return &e.rich.Entry
	}
	return &e.entry.Entry
}

func (e *textEdit) updateStatus() {
	entry := e.cursor()
	e.cursorRow.SetText(fmt.Sprintf("%d", entry.CursorRow+1))
	e.cursorCol.SetText(fmt.Sprintf("%d", entry.CursorColumn+1))
}

func (e *textEdit) typedShortcut(shortcut fyne.Shortcut) {
	if e.richMode {
		e.rich.TypedShortcut(shortcut)
		return
	}
	e.entry.TypedShortcut(shortcut)
}

func (e *textEdit) cut() {
	e.typedShortcut(&fyne.ShortcutCut{Clipboard: fyne.CurrentApp().Clipboard()})
}

func (e *textEdit) copy() {
	e.typedShortcut(&fyne.ShortcutCopy{Clipboard: fyne.CurrentApp().Clipboard()})
}

func (e *textEdit) paste() {
	e.typedShortcut(&fyne.ShortcutPaste{Clipboard: fyne.CurrentApp().Clipboard()})
}

// focus moves keyboard input to the entry for the current mode.
func (e *textEdit) focus() {
	if e.richMode {
		e.window.Canvas().Focus(e.rich)
		return
	}
	e.window.Canvas().Focus(e.entry)
}

// text returns the content to save, which is markdown when in rich mode.
func (e *textEdit) text() string {
	if !e.richMode {
		return e.entry.Text
	}

	markdown := e.rich.Markdown()
	if markdown != "" {
		markdown += "\n"
	}
	return markdown
}

// setText replaces the content without changing mode, parsing it as markdown when in rich mode.
func (e *textEdit) setText(text string) {
	if e.richMode {
		e.rich.ParseMarkdown(text)
		return
	}
	e.entry.SetText(text)
}

// setMode shows the editor for the requested mode and loads the content into it.
// In rich mode the content is parsed as markdown, otherwise it is used as it is.
func (e *textEdit) setMode(rich bool, text string) {
	e.richMode = rich
	e.setText(text)

	// the editor that is now hidden should not hold on to a copy of the document
	if rich {
		e.entry.SetText("")
		e.content.Objects = []fyne.CanvasObject{e.rich}
	} else {
		e.rich.SetText("")
		e.content.Objects = []fyne.CanvasObject{container.NewScroll(e.entry)}
	}
	e.content.Refresh()

	e.richToggle.setOn(rich)
	e.richItem.Checked = rich
	e.formatMenu.Refresh()
	e.updateStatus()
	e.focus()
}

// toggleRich switches between plain and rich text, keeping the current content.
// The styles of rich text become the markdown that describes them, which is parsed
// again when going back to rich text.
func (e *textEdit) toggleRich() {
	text := e.entry.Text
	if e.richMode {
		text = e.rich.Markdown()
	}
	e.setMode(!e.richMode, text)

	// plain text is saved as markdown once it has styles, a markdown file can be edited in either mode
	e.rename = e.richMode && e.uri != nil && !isMarkdown(e.uri)
	if text != "" || e.uri != nil {
		e.changed.Set(true)
	}
}

func (e *textEdit) buildToolbar(w fyne.Window) *widget.Toolbar {
	e.richToggle = newToolbarToggle(theme.DocumentIcon(), e.toggleRich)

	return widget.NewToolbar(
		widget.NewToolbarAction(theme.FolderOpenIcon(), e.open),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), e.save),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.FileIcon(), func() {
			dialog.ShowConfirm("Start a new document",
				"Are you sure you want to clear the contents of this editor?",
				func(ok bool) {
					if !ok {
						return
					}

					e.setText("")
					e.uri = nil
					e.rename = false
				}, w)
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ContentCutIcon(), e.cut),
		widget.NewToolbarAction(theme.ContentCopyIcon(), e.copy),
		widget.NewToolbarAction(theme.ContentPasteIcon(), e.paste),
		widget.NewToolbarSeparator(),
		e.richToggle,
	)
}

// makeUI loads a new text editor
func (e *textEdit) makeUI(w fyne.Window) fyne.CanvasObject {
	e.entry = newEditorEntry(e)
	e.rich = newRichEditorEntry(e)
	e.cursorRow = widget.NewLabel("1")
	e.cursorCol = widget.NewLabel("1")

	e.entry.OnCursorChanged = e.updateStatus
	e.rich.OnCursorChanged = e.updateStatus
	e.entry.OnChanged = func(s string) {
		e.changed.Set(true)
	}
	e.rich.OnChanged = e.entry.OnChanged

	e.richItem = fyne.NewMenuItem("Rich Text", e.toggleRich)
	e.formatMenu = fyne.NewMenu("Format", e.richItem)

	toolbar := e.buildToolbar(w)
	status := container.NewHBox(layout.NewSpacer(),
		widget.NewLabel("Cursor Row:"), e.cursorRow,
		widget.NewLabel("Col:"), e.cursorCol)
	e.content = container.NewStack(container.NewScroll(e.entry))
	return container.NewBorder(toolbar, status, nil, nil, e.content)
}

// typedSave handles the save shortcut for an editor, reporting whether it was used.
func (e *textEdit) typedSave(shortcut fyne.Shortcut) bool {
	desk, ok := shortcut.(*desktop.CustomShortcut)
	if !ok || desk.Modifier != fyne.KeyModifierControl || desk.KeyName != fyne.KeyS {
		return false
	}

	e.save()
	return true
}

// toolbarToggle is a toolbar item that is highlighted whilst the state it controls is on.
type toolbarToggle struct {
	button *widget.Button
}

func newToolbarToggle(icon fyne.Resource, onActivated func()) *toolbarToggle {
	button := widget.NewButtonWithIcon("", icon, onActivated)
	button.Importance = widget.LowImportance
	return &toolbarToggle{button: button}
}

func (t *toolbarToggle) ToolbarObject() fyne.CanvasObject {
	return t.button
}

func (t *toolbarToggle) setOn(on bool) {
	t.button.Importance = widget.LowImportance
	if on {
		t.button.Importance = widget.HighImportance
	}
	t.button.Refresh()
}

type editorEntry struct {
	widget.Entry

	edit *textEdit
}

func newEditorEntry(edit *textEdit) *editorEntry {
	e := &editorEntry{edit: edit}
	e.MultiLine = true
	e.ExtendBaseWidget(e)
	return e
}

func (e *editorEntry) TypedShortcut(shortcut fyne.Shortcut) {
	if e.edit.typedSave(shortcut) {
		return
	}

	e.Entry.TypedShortcut(shortcut)
}

type richEditorEntry struct {
	widget.RichTextEntry

	edit *textEdit
}

func newRichEditorEntry(edit *textEdit) *richEditorEntry {
	e := &richEditorEntry{edit: edit}
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.TypeMarkdown = true
	e.ExtendBaseWidget(e)
	return e
}

func (e *richEditorEntry) TypedShortcut(shortcut fyne.Shortcut) {
	if e.edit.typedSave(shortcut) {
		return
	}

	e.RichTextEntry.TypedShortcut(shortcut)
}
