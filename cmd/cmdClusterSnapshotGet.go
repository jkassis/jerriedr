package main

import (
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "clustersnapshotget",
		Short: "Retrieve a snapshot of cluster services and save to a local archive.",
		Long:  `This command is a shortcut for servicesnapshotcopy with several presets.`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDClusterSnapshotGet(v)
		},
	}

	// kube
	FlagsAddKubeFlags(c, v)

	// localDir
	MAIN.AddCommand(c)
}

func CMDClusterSnapshotGet(v *viper.Viper) {
	// dstArchiveSpec := "local|/var/cluster"

	// setup the srcArchiveSet
	srcArchiveSet := schema.ArchiveSetNew()
	for _, srcArchiveSpec := range []string{
		"statefulset|fg/dockie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/ledgie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/tickie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/dubbie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/keevie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/permie|/var/data/single/<pod>-server-0/backup",
	} {
		srcArchiveSet.ArchiveAdd(srcArchiveSpec)
	}

	// get a kube client
	kubeClient, kubeErr := KubeClientGet(v)
	if kubeErr != nil {
		core.Log.Fatalf("kube client initialization failed: %v", kubeErr)
	}

	// fetch all the files
	err := srcArchiveSet.FilesFetch(kubeClient)
	if err != nil {
		core.Log.Fatalf("failed to get files for cluster archive set: %v", err)
	}

	picker := ArchiveFileSetPickerNew()
	picker.ArchiveSetPut(srcArchiveSet)
	picker.Run()

	// table := tview.NewTable()
	// table.SetSelectable(true, false)

	// for i, s := range []string{
	// 	"Select",
	// 	"these",
	// 	"columns",
	// 	"while",
	// 	"typing",
	// 	"in",
	// 	"the",
	// 	"InputField",
	// } {
	// 	table.SetCellSimple(i, 0, s)
	// }

	// input := tview.NewInputField()
	// input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	// 	switch event.Key() {
	// 	case tcell.KeyUp, tcell.KeyDown:
	// 		table.InputHandler()(event, nil)
	// 	}

	// 	return event
	// })

	// Table for snapshots

	// // table.Select(0, 0).SetFixed(1, 1).SetDoneFunc(func(key tcell.Key) {
	// // 	if key == tcell.KeyEscape {
	// // 		app.Stop()
	// // 	}
	// // 	if key == tcell.KeyEnter {
	// // 		table.SetSelectable(true, true)
	// // 	}
	// // })

	// // table.SetSelectedFunc(func(row int, column int) {
	// // 	table.GetCell(row, column).SetTextColor(tcell.ColorRed)
	// // 	table.SetSelectable(false, false)
	// // })

	// // sss := srcArchiveSet.SnapshotSetGetNext()

	// for r := 0; r < rows; r++ {
	// 	for c := 0; c < cols; c++ {
	// 		color := tcell.ColorWhite
	// 		if c < 1 || r < 1 {
	// 			color = tcell.ColorYellow
	// 		}
	// 		table.SetCell(r, c,
	// 			tview.NewTableCell(lorem[word]).
	// 				SetTextColor(color).
	// 				SetAlign(tview.AlignCenter))
	// 		word = (word + 1) % len(lorem)
	// 	}
	// }

	// if err := app.SetRoot(flex, true).SetFocus(table).Run(); err != nil {
	// 	panic(err)
	// }

	// // box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
	// // inputField := tview.NewInputField().
	// // 	SetLabel("Enter a number: ").
	// // 	SetPlaceholder("E.g. 1234").
	// // 	SetFieldWidth(10).
	// // 	SetAcceptanceFunc(tview.InputFieldInteger).
	// // 	SetDoneFunc(func(key tcell.Key) {
	// // 		app.Stop()
	// // 	})

	// // {
	// // 	core.Log.Warnf("CMDClusterSnapshotGet: starting")
	// // 	start := time.Now()
	// // 	errGroup := errgroup.Group{}
	// // 	for _, srcArchiveSpec := range srcArchiveSpecs {
	// // 		srcArchiveSpec := srcArchiveSpec
	// // 		errGroup.Go(func() error {
	// // 			return SnapshotCopy(v, srcArchiveSpec, dstArchiveSpec)
	// // 		})
	// // 	}

	// // 	err := errGroup.Wait()
	// // 	if err != nil {
	// // 		core.Log.Error(err)
	// // 	}

	// // 	duration := time.Since(start)
	// // 	core.Log.Warnf("CMDClusterSnapshotGet: took %s", duration.String())
	// // }
}
