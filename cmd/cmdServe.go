package main

import (
	"fmt"
	"net/http"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	core.Log.Warnf("defining CMDServe")
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "serve",
		Short: "Serves liveness for jerrie services.",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDServe(v)
		},
	}

	MAIN.AddCommand(c)
}

func CMDServe(v *viper.Viper) {
	fmt.Printf("Serving liveness on 10000")
	statusHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("T"))
	}
	http.HandleFunc("/statusAlive", statusHandler)
	http.ListenAndServe(":10000", nil)
}
