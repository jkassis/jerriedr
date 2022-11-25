package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "prodsnapshotget",
		Short: "Retrieve a snapshot of cluster services and save to a local archive.",
		Long:  `This command is a shortcut for servicesnapshotcopy with several presets.`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDProdSnapshotGet(v)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}

func CMDProdSnapshotGet(v *viper.Viper) {
	srcArchiveSpecs := prodArchiveSpecs
	dstArchiveSpecs := prodRepoArchiveSpecs
	CMDEnvSnapshotGet(v, srcArchiveSpecs, dstArchiveSpecs)
}
