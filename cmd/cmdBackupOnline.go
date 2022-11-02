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
		Use:   "backuponline",
		Short: "Backup the DB to the permanent DB data backup directory",
		Long: `If you have trouble making this work, this command is equivalent to the following curl call...

curl -d '{ "UUID": "<UUID>", "Fn": "/v1/Backup", "Body": {} }' -H 'Content-Type: application/json' http://<hostport>/raft/leader/read

eg..
curl -d '{ "UUID": "9db4caec-a449-4082-a1c3-ac82b4d25444", "Fn": "/v1/Backup", "Body": {} }' -H 'Content-Type: application/json' http://dockie-0.dockie-int.fg.svc.cluster.local:10000/raft/leader/read
`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDBackupOnline(v)
		},
	}

	CMDServerConfig(c, v)
	MAIN.AddCommand(c)
}

func CMDBackupOnline(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("BackupOnline: starting")

	hostport := v.GetString(FLAG_SERVER_HOSTPORT)
	scheme := v.GetString(FLAG_SERVER_SCHEME)
	reqBody := strings.NewReader(fmt.Sprintf(`
	{
	  "UUID": "%s",
	  "Fn": "/v1/Backup",
	  "Body": {}
	}`, uuid.NewString()))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s/raft/leader/read", scheme, hostport), reqBody)
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

	core.Log.Warnf("BackupOnline: %d %s", resp.StatusCode, body)
	if err != nil {
		core.Log.Fatalln(err)
	}

	duration := time.Since(start)
	core.Log.Warnf("BackupOnline: took %s", duration.String())
}
