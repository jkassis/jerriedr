package ui

import (
	"fmt"
	"time"

	"github.com/rivo/tview"
)

type Watch struct {
	Progress int64
	Total    int64
	Item     string
	Unit     string
}

type ProgressWatcher struct {
	App          *tview.Application
	ProgressBars *tview.Table
	RootView     *tview.Table
	Watches      []*Watch
}

func ProgressWatcherNew() *ProgressWatcher {
	p := &ProgressWatcher{
		Watches: make([]*Watch, 0),
	}

	p.ProgressBars = tview.NewTable()
	p.ProgressBars.SetSelectable(false, false)
	p.ProgressBars.SetBorders(false).SetBorder(true).SetTitle("Progress")

	p.RootView = p.ProgressBars

	p.App = tview.NewApplication()
	p.App.SetRoot(p.RootView, true).SetFocus(p.RootView)
	return p
}

func (p *ProgressWatcher) AddWatch(watch *Watch) func(progress int64) {
	cell := tview.NewTableCell(fmt.Sprintf("Hello %d", len(p.Watches)))
	p.ProgressBars.SetCell(len(p.Watches), 0, cell)
	p.Watches = append(p.Watches, watch)

	return func(progress int64) {
		watch.Progress += progress
	}
}

func (p *ProgressWatcher) Run() {
	stopCh := make(chan struct{})

	go func() {
		updateWatches := func() {
			for i, watch := range p.Watches {
				message := fmt.Sprintf("[ %12d of %12d %s ] %s", watch.Progress, watch.Total, watch.Unit, watch.Item)
				p.ProgressBars.GetCell(i, 0).SetText(message)
			}
		}

		tick := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-stopCh:
				return
			case <-tick.C:
				p.App.QueueUpdateDraw(updateWatches)
			}
		}
	}()

	if err := p.App.Run(); err != nil {
		panic(err)
	}
	stopCh <- struct{}{}
}
