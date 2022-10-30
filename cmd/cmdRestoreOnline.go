package main

import (
	"fmt"
	"io"
	"time"

	"net/http"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	core.Log.Warnf("defining CMDRestoreOnline")
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "restoreonline",
		Short: "Restore the DB from the permanent DB data restore directory",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDRestoreOnline(v)
		},
	}

	CMDServerConfig(c, v)
	MAIN.AddCommand(c)
}

func CMDRestoreOnline(v *viper.Viper) {
	start := time.Now()
	core.Log.Warn("Restore: starting")

	hostport := v.GetString(FLAG_SERVER_HOSTPORT)
	scheme := v.GetString(FLAG_SERVER_SCHEME)
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s://%s/v1/Restore", scheme, hostport), nil)
	if err != nil {
		core.Log.Fatalln(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		core.Log.Fatalln(err)
	}

	core.Log.Warnf("Restore: got response")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		core.Log.Fatalln(err)
	}

	core.Log.Warnf("Restore: %s", body)
	if err != nil {
		core.Log.Fatalln(err)
	}

	duration := time.Since(start)
	core.Log.Warnf("Restore: took %s", duration.String())
}
