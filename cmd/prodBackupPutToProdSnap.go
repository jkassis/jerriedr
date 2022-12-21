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
		Use:   "prodBackupToProdSnap",
		Short: ``,
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			CMDProdBackupToProdSnap(v)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}

func CMDProdBackupToProdSnap(v *viper.Viper) {
	srcArchiveSpecs := prodBackupArchiveSpecs
	dstArchiveSpecs := prodSnapArchiveSpecs
	EnvCopy(v, srcArchiveSpecs, dstArchiveSpecs)
}
