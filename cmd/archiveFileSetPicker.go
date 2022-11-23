package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/rivo/tview"
)

type ArchiveFileSetPicker struct {
	ArchiveSet                     *schema.ArchiveSet
	App                            *tview.Application
	SelectedSnapshotArchiveFileSet *schema.ArchiveFileSet
	SelectedSnapshotFilesView      *tview.Table
	SelectedSnapshotStatusView     *tview.Table
	FiltersView                    *tview.TextView
	SnapshotsView                  *tview.Table
	RootView                       *tview.Flex
}

func ArchiveFileSetPickerNew() *ArchiveFileSetPicker {
	p := &ArchiveFileSetPicker{}

	p.SnapshotsView = tview.NewTable()
	p.SnapshotsView.SetSelectable(true, false)
	p.SnapshotsView.SetSelectedFunc(p.Select)
	p.SnapshotsView.SetSelectionChangedFunc(p.SelectedSnapshotViewRender)
	p.SnapshotsView.SetBorders(false).SetBorder(true).SetTitle("Snapshots")

	p.FiltersView = tview.NewTextView()
	p.FiltersView.SetBorder(true).SetTitle("Filters")

	p.SelectedSnapshotStatusView = tview.NewTable()
	p.SelectedSnapshotStatusView.SetSelectable(false, false)
	p.SelectedSnapshotStatusView.SetBorders(false).SetBorder(true).SetTitle("Status")

	p.SelectedSnapshotFilesView = tview.NewTable()
	p.SelectedSnapshotFilesView.SetSelectable(false, false).SetSeparator(tview.Borders.Vertical)
	p.SelectedSnapshotFilesView.SetBorders(false).SetBorder(true).SetTitle("Files")

	selectedSnapshotView := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.SelectedSnapshotStatusView, 0, 1, false).
		AddItem(p.SelectedSnapshotFilesView, 0, 10, false)
	selectedSnapshotView.SetBorder(true).SetTitle("Selected Snapshot")

	p.RootView = tview.NewFlex().SetDirection(tview.FlexColumn).
		// AddItem(tview.NewBox().SetBorder(true).SetTitle("Left (1/2 x width of Top)"), 0, 1, false).
		AddItem(p.SnapshotsView, 0, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(p.FiltersView, 0, 1, false).
			AddItem(selectedSnapshotView, 0, 10, false),
			0, 3, false)

	p.App = tview.NewApplication()
	p.App.SetRoot(p.RootView, true)
	return p
}

func (p *ArchiveFileSetPicker) SelectedSnapshotViewRender(row, col int) {
	selectedCell := p.SnapshotsView.GetCell(row, col)
	ref := selectedCell.GetReference()
	archiveFileSet := ref.(*schema.ArchiveFileSet)

	// Update SelectedSnapshotStatusView
	{
		// default state of snapshot cell is "looks good"
		p.SelectedSnapshotStatusView.Clear()
		p.SelectedSnapshotStatusView.SetBackgroundColor(tcell.ColorGreen)
		cell := tview.NewTableCell("looks good")
		p.SelectedSnapshotStatusView.SetCell(0, 0, cell)

		// get all status messages
		var isInError, isInWarn bool
		var statusMessageCount int

		// validate number of files
		missingArchiveFilesNum := len(p.ArchiveSet.Archives) - len(archiveFileSet.ArchiveFiles)
		if missingArchiveFilesNum > 0 {
			cell := tview.NewTableCell(fmt.Sprintf("error: missing %d archive files", missingArchiveFilesNum))
			p.SelectedSnapshotStatusView.SetCell(statusMessageCount, 0, cell)
			statusMessageCount++
			isInError = true
		}

		// validate timestamps
		first, last := archiveFileSet.FirstAndLastArchiveFileTime()
		if last.Sub(first) > time.Second {
			cell := tview.NewTableCell("warning: expected all file timestamps to be < 1 sec apart")
			p.SelectedSnapshotStatusView.SetCell(statusMessageCount, 0, cell)

			statusMessageCount++
			isInWarn = true
		}

		// set the background color
		if isInWarn {
			p.SelectedSnapshotStatusView.SetBackgroundColor(tcell.ColorYellow)
		}
		if isInError {
			p.SelectedSnapshotStatusView.SetBackgroundColor(tcell.ColorRed)
		}
	}

	// Update SelectedSnapshotFilesView
	{
		// copy and sort archives by name
		archives := make([]*schema.Archive, len(p.ArchiveSet.Archives))
		copy(archives, p.ArchiveSet.Archives)
		sort.Sort(schema.BySpec(archives))

		// for each archive in the archiveSet
		p.SelectedSnapshotFilesView.Clear()
		for r, archive := range archives {
			// get the archiveFile
			var timestamp string
			archiveSpecFilePath := archive.Spec
			for _, archiveFile := range archiveFileSet.ArchiveFiles {
				if archiveFile.Archive == archive || archiveFile.Archive.Parent == archive {
					timestamp = archiveFile.Time.Format(time.UnixDate)
					archiveSpecFilePath += "/" + archiveFile.Name
					break
				}
			}

			timeCell := tview.NewTableCell(timestamp).SetTextColor(tcell.ColorBlue)
			p.SelectedSnapshotFilesView.SetCell(r, 0, timeCell.SetAlign(tview.AlignLeft))

			specCell := tview.NewTableCell(archiveSpecFilePath).SetTextColor(tcell.ColorBlue)
			p.SelectedSnapshotFilesView.SetCell(r, 1, specCell.SetAlign(tview.AlignLeft))
		}
	}
}

func (p *ArchiveFileSetPicker) Select(row, col int) {
	selectedCell := p.SnapshotsView.GetCell(row, col)
	ref := selectedCell.GetReference()
	p.SelectedSnapshotArchiveFileSet = ref.(*schema.ArchiveFileSet)
	p.App.Stop()
	core.Log.Warnf("retrieving snapshot from '%s'", selectedCell.Text)
}

func (p *ArchiveFileSetPicker) ArchiveSetPut(as *schema.ArchiveSet) {
	p.ArchiveSet = as

	// add 1 row per snapshot
	as.SeekTo(time.Now())
	for r := 0; true; r++ {
		archiveFileSet := as.ArchiveFileSetGetNext()
		if archiveFileSet == nil {
			break
		}
		_, last := archiveFileSet.FirstAndLastArchiveFileTime()
		cell := tview.NewTableCell(
			last.Format(time.UnixDate)).
			SetReference(archiveFileSet).
			SetTextColor(tcell.ColorBlue)
		p.SnapshotsView.SetCell(r, 0, cell.SetAlign(tview.AlignCenter))
	}

	p.SelectedSnapshotViewRender(0, 0)
}

func (p *ArchiveFileSetPicker) Run() {
	if err := p.App.Run(); err != nil {
		panic(err)
	}
}
