package main

import (
	"os"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	core.Log.Warnf("defining CMDRestore")
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "restore",
		Short: "Restore the DB from STDIN",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDRestore(v)
		},
	}

	CMDDBConfig(c, v)
	MAIN.AddCommand(c)
}

func CMDRestore(v *viper.Viper) {
	dbBadger := CMDDBRun(v)

	start := time.Now()
	core.Log.Warn("Restore: starting")
	err := dbBadger.DB.Load(os.Stdin, 256)
	if err != nil {
		core.Log.Error(err)
		return
	}
	duration := time.Since(start)
	core.Log.Warnf("Restore: took %s", duration.String())
}
