package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"net/http"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	core.Log.Warnf("defining CMDBackupOnline")
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "backuponline",
		Short: "Backup the DB to the permanent DB data backup directory",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDBackupOnline(v)
		},
	}

	CMDServerConfig(c, v)
	MAIN.AddCommand(c)
}

func CMDBackupOnline(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("Backup: starting")

	hostport := v.GetString(FLAG_SERVER_HOSTPORT)
	scheme := v.GetString(FLAG_SERVER_SCHEME)

	// export interface RPCReq {
	//   UUID: UUID
	//   Fn: string
	//   Body: object
	// }
	reqBody := strings.NewReader(`
	{
		UUID: '',
	  Fn: 'v1/Backup',
	  Body: { },
	}`)

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s://%s/raft/leader/read", scheme, hostport), reqBody)
	if err != nil {
		core.Log.Fatalln(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		core.Log.Fatal(err)
	}

	core.Log.Warnf("BackupOnline: got response")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		core.Log.Fatalln(err)
	}

	core.Log.Warnf("BackupOnline: %s", body)
	if err != nil {
		core.Log.Fatalln(err)
	}

	duration := time.Since(start)
	core.Log.Warnf("BackupOnline: took %s", duration.String())
}
