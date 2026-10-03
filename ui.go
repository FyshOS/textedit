package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/x/fyne/menu"
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
	// richView holds the rich editor below the bar of styles that can be applied to it
	richView   *fyne.Container
	formatMenu *fyne.Menu

	uri fyne.URI
	// rename is set when a file that is not markdown was switched to rich mode,
	// so the next save asks for a name with the markdown extension
	rename  bool
	recents *menu.Recents
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
		e.content.Objects = []fyne.CanvasObject{e.richView}
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

// buildStyleBar creates the bar that applies the styles which rich text can be saved with.
func (e *textEdit) buildStyleBar() fyne.CanvasObject {
	quote := widget.RichTextStyleBlockquote
	quote.QuotingDepth = 1

	style := func(icon fyne.Resource, apply func()) fyne.CanvasObject {
		button := widget.NewButtonWithIcon("", theme.NewThemedResource(icon), apply)
		button.Importance = widget.LowImportance
		return button
	}

	// the bar scrolls so that a narrow window is not forced wider by it
	return container.NewHScroll(container.NewHBox(
		style(resourceFormatclearSvg, e.clearStyle),
		style(resourceFormath1Svg, e.lineStyle(widget.RichTextStyleHeading)),
		style(resourceFormath2Svg, e.lineStyle(widget.RichTextStyleSubHeading)),
		style(resourceFormatquoteSvg, e.lineStyle(quote)),
		widget.NewSeparator(),
		style(resourceFormatboldSvg, e.toggleStyle(fyne.KeyB, 0)),
		style(resourceFormatitalicSvg, e.toggleStyle(fyne.KeyI, 0)),
		style(resourceFormatstrikethroughSvg, e.toggleStyle(fyne.KeyX, fyne.KeyModifierShift)),
		style(resourceCodeSvg, e.selectionStyle(widget.RichTextStyleCodeInline)),
		widget.NewSeparator(),
		style(resourceFormatlistbulletedSvg, e.lineBlock(listItem(false))),
		style(resourceFormatlistnumberedSvg, e.lineBlock(listItem(true))),
		style(resourceCodeblocksSvg, e.lineBlock(codeBlock)),
	))
}

// lineBlock returns an action that moves the selected lines, or the line the cursor
// is on, into a block such as a list or a code block by rebuilding the segments around them.
func (e *textEdit) lineBlock(wrap func(before, lines, after []widget.RichTextSegment) []widget.RichTextSegment) func() {
	return func() {
		defer e.focus()

		start, end := e.selectedLines()
		before, lines, after, ok := splitLines(e.rich.Segments(), start, end, end < utf8.RuneCountInString(e.rich.Text))
		if !ok {
			return // the lines are already inside a block
		}

		e.rebuild(wrap(before, lines, after))
	}
}

// rebuild replaces the segments of the rich text with ones that hold the same text
// in different blocks, so the cursor can go back to where it was.
func (e *textEdit) rebuild(segments []widget.RichTextSegment) {
	row, col := e.rich.CursorRow, e.rich.CursorColumn
	e.rich.SetSegments(segments)
	e.rich.CursorRow, e.rich.CursorColumn = row, col
	e.rich.Refresh()
	e.changed.Set(true)
}

// listItem returns a wrap function that makes each line an item of a list, joining
// a list of the same kind on either side of them.
func listItem(ordered bool) func(before, lines, after []widget.RichTextSegment) []widget.RichTextSegment {
	return func(before, lines, after []widget.RichTextSegment) []widget.RichTextSegment {
		var items []widget.RichTextSegment
		for _, line := range splitAtLineBreaks(lines) {
			items = append(items, &widget.ParagraphSegment{Texts: line})
		}

		if n := len(before); n > 0 {
			if list, ok := before[n-1].(*widget.ListSegment); ok && list.Ordered == ordered {
				list.Items = append(list.Items, items...)
				return append(before, after...)
			}
		}
		if len(after) > 0 {
			if list, ok := after[0].(*widget.ListSegment); ok && list.Ordered == ordered {
				list.Items = append(items, list.Items...)
				return append(before, after...)
			}
		}

		list := &widget.ListSegment{Items: items, Ordered: ordered}
		return append(append(before, list), after...)
	}
}

// splitAtLineBreaks divides inline segments into one list of segments per line,
// with the line breaks removed.
func splitAtLineBreaks(segments []widget.RichTextSegment) [][]widget.RichTextSegment {
	lines := [][]widget.RichTextSegment{{}}
	for _, seg := range segments {
		text, ok := seg.(*widget.TextSegment)
		if !ok {
			lines[len(lines)-1] = append(lines[len(lines)-1], seg)
			continue
		}

		for i, part := range strings.Split(text.Text, "\n") {
			if i > 0 {
				lines = append(lines, []widget.RichTextSegment{})
			}
			if part != "" {
				lines[len(lines)-1] = append(lines[len(lines)-1], &widget.TextSegment{Style: text.Style, Text: part})
			}
		}
	}
	return lines
}

// codeBlock is a wrap function that makes the lines a code block, joining a code block right before them.
func codeBlock(before, lines, after []widget.RichTextSegment) []widget.RichTextSegment {
	text := &strings.Builder{}
	for _, seg := range lines {
		text.WriteString(seg.Textual())
	}

	if n := len(before); n > 0 {
		if code, ok := before[n-1].(*widget.CodeBlockSegment); ok {
			code.Text += text.String() + "\n"
			return append(before, after...)
		}
	}

	block := &widget.CodeBlockSegment{Text: text.String()}
	return append(append(before, block), after...)
}

// selectedLines returns the rune offsets of the start and end of the lines that the
// selection covers, or of the line that the cursor is on when nothing is selected.
// The selection lies to one side of the cursor, which is found by checking whether
// its last line is the text that comes before the cursor on its line.
func (e *textEdit) selectedLines() (start, end int) {
	text := []rune(e.rich.Text)
	pos := min(e.rich.CursorTextOffset(), len(text))
	start, end = lineAround(text, pos)

	selected := e.rich.SelectedText()
	breaks := strings.Count(selected, "\n")
	if breaks == 0 {
		return start, end
	}

	last := selected[strings.LastIndex(selected, "\n")+1:]
	if utf8.RuneCountInString(trimListMarker(last)) == pos-start {
		for ; breaks > 0 && start > 0; breaks-- {
			start, _ = lineAround(text, start-1)
		}
	} else {
		for ; breaks > 0 && end < len(text); breaks-- {
			_, end = lineAround(text, end+1)
		}
	}
	return start, end
}

// trimListMarker removes the bullet or number that selected text includes at the
// start of a list item, as it is not part of the content.
func trimListMarker(line string) string {
	trimmed := strings.TrimLeft(line, " ")
	if strings.HasPrefix(trimmed, "\u2022 ") {
		return trimmed[len("\u2022 "):]
	}

	digits := strings.TrimLeft(trimmed, "0123456789")
	if len(digits) < len(trimmed) && strings.HasPrefix(digits, ". ") {
		return digits[2:]
	}
	return line
}

// lineAround returns the rune offsets of the start and end of the line holding
// the given offset, not including the line break that ends it.
func lineAround(text []rune, pos int) (start, end int) {
	start = min(pos, len(text))
	end = start
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	for end < len(text) && text[end] != '\n' {
		end++
	}
	return start, end
}

// splitLines divides the top level segments into those before the lines between the
// rune offsets, those that make up the lines and those after them. The line break
// that ends the last line is left out when there is one, as a block adds its own.
// It reports false if the lines are held inside a block, which can not be split.
func splitLines(segments []widget.RichTextSegment, start, end int, lineBreak bool) (before, lines, after []widget.RichTextSegment, ok bool) {
	next := end // where the content after the line starts
	if lineBreak {
		next++
	}

	lines = []widget.RichTextSegment{}
	off := 0
	for _, seg := range segments {
		segStart := off
		off += segmentLength(seg)

		text, isText := seg.(*widget.TextSegment)
		switch {
		case off <= start && (off < start || segmentLength(seg) > 0):
			before = append(before, seg)
		case segStart >= next:
			after = append(after, seg)
		case !isText && segStart >= start && off <= end:
			lines = append(lines, seg)
		case !isText:
			return nil, nil, nil, false
		default:
			runes := []rune(text.Text)
			cut := func(from, to int) {
				from, to = max(from, segStart), min(to, off)
				if from >= to {
					return
				}
				piece := &widget.TextSegment{Style: text.Style, Text: string(runes[from-segStart : to-segStart])}
				switch {
				case to <= start:
					before = append(before, piece)
				case from >= next:
					after = append(after, piece)
				default:
					lines = append(lines, piece)
				}
			}
			cut(segStart, start)
			cut(start, end)
			cut(next, off)
		}
	}
	return before, lines, after, true
}

// segmentLength returns the number of runes of text that a segment holds, including those of any blocks inside it.
func segmentLength(seg widget.RichTextSegment) int {
	if _, ok := seg.(*widget.CodeBlockSegment); !ok {
		if block, ok := seg.(interface {
			Segments() []widget.RichTextSegment
		}); ok {
			n := 0
			for _, child := range block.Segments() {
				n += segmentLength(child)
			}
			return n
		}
	}
	return utf8.RuneCountInString(seg.Textual())
}

// toggleStyle returns an action that turns a style on or off, for the selection or the text typed next.
// It uses the shortcut of the rich text entry, which keeps the other styles of the text it changes.
func (e *textEdit) toggleStyle(key fyne.KeyName, modifier fyne.KeyModifier) func() {
	return func() {
		e.rich.TypedShortcut(&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierShortcutDefault | modifier})
		e.focus()
	}
}

// selectionStyle returns an action that sets the style of the selected text.
func (e *textEdit) selectionStyle(style widget.RichTextStyle) func() {
	return func() {
		if e.rich.SelectedText() != "" {
			e.rich.SetStyleForSelection(style)
			e.changed.Set(true)
		}
		e.focus()
	}
}

// lineStyle returns an action that sets the style of the selected lines, or the
// line that the cursor is on, for the styles that apply to a whole line.
func (e *textEdit) lineStyle(style widget.RichTextStyle) func() {
	return func() {
		start, end := e.selectedLines()
		if start < end {
			e.rich.SetStyleForRange(start, end, style)
			e.changed.Set(true)
		}
		e.focus()
	}
}

// clearStyle removes the styles of the selected text when the selection is within
// a line. Otherwise it makes the selected lines, or the line that the cursor is on,
// plain text by taking them out of any list or code block and clearing their styles.
func (e *textEdit) clearStyle() {
	defer e.focus()

	if selected := e.rich.SelectedText(); selected != "" && !strings.Contains(selected, "\n") {
		e.rich.SetStyleForSelection(widget.RichTextStyleInline)
		e.changed.Set(true)
		return
	}

	start, end := e.selectedLines()
	if segments, changed := unwrapLines(e.rich.Segments(), start, end); changed {
		e.rebuild(segments)
	}

	if start < end {
		e.rich.SetStyleForRange(start, end, widget.RichTextStyleInline)
		e.changed.Set(true)
	}
}

// unwrapLines takes the lines between the rune offsets out of the lists and code
// blocks that hold them, keeping the other lines of those blocks in blocks of their
// own. It reports whether any of the segments were changed.
func unwrapLines(segments []widget.RichTextSegment, start, end int) ([]widget.RichTextSegment, bool) {
	var out []widget.RichTextSegment
	changed := false

	off := 0
	for _, seg := range segments {
		segStart := off
		off += segmentLength(seg)
		if off < start || segStart > end {
			out = append(out, seg)
			continue
		}

		switch block := seg.(type) {
		case *widget.ListSegment:
			out = append(out, unwrapList(block, start, end, segStart)...)
			changed = true
		case *widget.CodeBlockSegment:
			out = append(out, unwrapCode(block, start, end, segStart)...)
			changed = true
		default:
			out = append(out, seg)
		}
	}
	return out, changed
}

// unwrapList returns the content of the list with the items that start on the lines
// between the rune offsets as plain segments. The remaining items stay in lists, with
// the first of those keeping the number that the list started from.
func unwrapList(list *widget.ListSegment, start, end, off int) []widget.RichTextSegment {
	var out []widget.RichTextSegment
	var kept *widget.ListSegment

	for i, item := range list.Items {
		itemStart := off
		off += segmentLength(item)

		if itemStart < start || itemStart > end {
			if kept == nil {
				kept = &widget.ListSegment{Ordered: list.Ordered}
				if i == 0 {
					kept.SetStartNumber(list.StartNumber())
				}
				out = append(out, kept)
			}
			kept.Items = append(kept.Items, item)
			continue
		}

		kept = nil
		switch content := item.(type) {
		case *widget.ParagraphSegment:
			out = append(out, content.Texts...)
		case *widget.ListSegment:
			out = append(out, unwrapList(content, start, end, itemStart)...)
		default:
			out = append(out, item)
		}
	}
	return out
}

// unwrapCode returns the lines of the code block that lie between the rune offsets
// as plain text, with the other lines kept in code blocks of their own.
func unwrapCode(code *widget.CodeBlockSegment, start, end, off int) []widget.RichTextSegment {
	var out []widget.RichTextSegment
	var kept *widget.CodeBlockSegment

	for _, line := range strings.SplitAfter(strings.TrimSuffix(code.Text, "\n"), "\n") {
		lineStart := off
		off += utf8.RuneCountInString(line)
		if !strings.HasSuffix(line, "\n") {
			line += "\n" // every line of a block is closed
		}

		if lineStart < start || lineStart > end {
			if kept == nil {
				kept = &widget.CodeBlockSegment{}
				out = append(out, kept)
			}
			kept.Text += line
			continue
		}

		kept = nil
		out = append(out, &widget.TextSegment{Style: widget.RichTextStyleInline, Text: line})
	}
	return out
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
	e.richView = container.NewBorder(e.buildStyleBar(), nil, nil, nil, e.rich)
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
