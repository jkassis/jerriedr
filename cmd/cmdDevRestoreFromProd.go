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
		Use:   "devservicerestorefromprodarch",
		Short: "Clear and load a dev service from a snapshot of prod services within a prod archive.",
		Long:  "Clear a dev monoservice and load data from an archive containing a snapshot of prod microservices.",
		Run: func(cmd *cobra.Command, args []string) {
			CMDDevServiceRestoreFromProd(v)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}

func CMDDevServiceRestoreFromProd(v *viper.Viper) {
	dstArchiveSpecs := devFromProdArchiveSpecs
	dstServiceSpecs := devFromProdServiceSpecs
	srcArchiveSpecs := localProdArchiveSpecs
	EnvRestore(v, srcArchiveSpecs, dstArchiveSpecs, dstServiceSpecs)
}
