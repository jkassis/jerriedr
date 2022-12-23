package main

import (
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/jkassis/jerriedr/cmd/ui"
	archive "github.com/jkassis/jerriedr/cmd/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	v := viper.New()

	c := &cobra.Command{
		Use:   "archviefilecopy",
		Short: "Copies snapshots from one archive to another. Archives can be in kube, on hosts, or local.",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDArchiveFileCopy(v)
		},
	}

	// flag configuration
	FlagsAddKubeFlags(c, v)
	FlagsAddSrcFlag(c, v)
	FlagsAddDstFlag(c, v)

	MAIN.AddCommand(c)
}

func CMDArchiveFileCopy(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("archiveFileCopy: starting")

	// get the archive files
	var srcArchiveFile, dstArchiveFile *schema.ArchiveFile
	{
		srcArchiveFileSpec := v.GetString(FLAG_SRC)
		srcArchiveFile = &schema.ArchiveFile{}
		err := srcArchiveFile.Parse(srcArchiveFileSpec)
		if err != nil {
			core.Log.Fatalf("archiveFileCopy: %v", err)
		}
	}
	{
		dstArchiveFileSpec := v.GetString(FLAG_DST)
		dstArchiveFile = &schema.ArchiveFile{}
		err := dstArchiveFile.Parse(dstArchiveFileSpec)
		if err != nil {
			core.Log.Fatalf("archiveFileCopy: %v", err)
		}
	}

	progressWatcher := ui.ProgressWatcherNew()
	go progressWatcher.Run()

	kubeClient, kubeErr := KubeClientGet(v)
	if kubeErr != nil {
		core.Log.Errorf("could not get KubeClient: %v", kubeErr)
	}

	archive.ArchiveFileCopy(kubeClient, srcArchiveFile, dstArchiveFile, progressWatcher)

	duration := time.Since(start)
	core.Log.Warnf("archiveFileCopy: took %s", duration.String())
}
