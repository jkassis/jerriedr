package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"net/http"

	"github.com/google/uuid"
	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "servicerestore",
		Short: "Restore the DB from the permanent DB data restore directory",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDServiceRestore(v)
		},
	}

	CMDServerConfig(c, v)
	CMDKubeConfig(c, v)
	MAIN.AddCommand(c)
}

func CMDServiceRestore(v *viper.Viper) {
	start := time.Now()
	core.Log.Warn("RestoreOnline: starting")

	hostport := v.GetString(FLAG_HOSTPORT)
	scheme := v.GetString(FLAG_PROTOCOL)
	reqBody := strings.NewReader(fmt.Sprintf(` { "UUID": "%s", "Fn": "/v1/Restore", "Body": {} }`, uuid.NewString()))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s/raft/leader/write", scheme, hostport), reqBody)
	if err != nil {
		core.Log.Fatalln(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		core.Log.Fatalln(err)
	}

	core.Log.Warnf("RestoreOnline: got response")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		core.Log.Fatalln(err)
	}

	core.Log.Warnf("RestoreOnline: %s", body)
	if err != nil {
		core.Log.Fatalln(err)
	}

	duration := time.Since(start)
	core.Log.Warnf("RestoreOnline: took %s", duration.String())
}
