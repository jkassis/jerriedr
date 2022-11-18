package main

import (
	"os"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "badgersnapshotput",
		Short: "Restore a snapshot of a Badger DB from STDIN",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDBadgerRestore(v)
		},
	}

	FlagsAddDBFlags(c, v)
	MAIN.AddCommand(c)
}

func CMDBadgerRestore(v *viper.Viper) {
	dbBadger := CMDDBRun(v)

	start := time.Now()
	core.Log.Warn("BadgerRestore: starting")
	err := dbBadger.DB.Load(os.Stdin, 256)
	if err != nil {
		core.Log.Error(err)
		return
	}
	duration := time.Since(start)
	core.Log.Warnf("BadgerRestore: took %s", duration.String())
}
