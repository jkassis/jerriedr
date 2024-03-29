package main

import (
	"fmt"
	"os"

	_ "embed"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// MAIN represents the base command when called without any subcommands
var MAIN = &cobra.Command{
	Use:   "jerriedr",
	Short: "A CLI for operations on jerrie services.",
}

func main() {
	err := MAIN.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

const (
	FLAG_KUBE_MASTER_URL  = "ku"
	FLAG_KUBE_CONFIG_PATH = "kc"
	FLAG_DB_DIR           = "db"
	FLAG_HOSTPORT         = "hp"
	FLAG_PROTOCOL         = "pr"
	FLAG_VERSION          = "vr"
	FLAG_SRC              = "sa"
	FLAG_DST              = "da"
	FLAG_SERVICE          = "se"
	FLAG_RESTORE_ARCHIVE  = "ra"
)

func FlagsAddDBFlags(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_DB_DIR, "", "database dir")
	c.MarkPersistentFlagRequired(FLAG_DB_DIR)
	v.BindPFlag(FLAG_DB_DIR, c.PersistentFlags().Lookup(FLAG_DB_DIR))
}

func FlagsAddKubeFlags(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_KUBE_CONFIG_PATH, "", "absolute path to the kubernetes config file")
	// c.MarkPersistentFlagRequired(FLAG_KUBE)
	v.BindPFlag(FLAG_KUBE_CONFIG_PATH, c.PersistentFlags().Lookup(FLAG_KUBE_CONFIG_PATH))

	c.PersistentFlags().String(FLAG_KUBE_MASTER_URL, "https://api.live.shinetribe.media:6443", "URL to the kubernetes master")
	// c.MarkPersistentFlagRequired(FLAG_KUBE)
	v.BindPFlag(FLAG_KUBE_MASTER_URL, c.PersistentFlags().Lookup(FLAG_KUBE_MASTER_URL))
}

func FlagsAddHostFlags(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_HOSTPORT, "localhost:10000", "server hostport")
	// c.MarkPersistentFlagRequired(FLAG_SERVER_HOSTPORT)
	v.BindPFlag(FLAG_HOSTPORT, c.PersistentFlags().Lookup(FLAG_HOSTPORT))
}

func FlagsAddProtocolFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_PROTOCOL, "http", "protocol: http | https")
	// c.MarkPersistentFlagRequired(FLAG_PROTOCOL)
	v.BindPFlag(FLAG_PROTOCOL, c.PersistentFlags().Lookup(FLAG_PROTOCOL))
}

func FlagsAddAPIVersionFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_VERSION, "v1", "backup protocol version")
	v.BindPFlag(FLAG_VERSION, c.PersistentFlags().Lookup(FLAG_VERSION))
}

func FlagsAddSrcFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_SRC, "", "source")
	c.MarkPersistentFlagRequired(FLAG_SRC)
	v.BindPFlag(FLAG_SRC, c.PersistentFlags().Lookup(FLAG_SRC))
}

func FlagsAddDstFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_DST, "", "destination")
	c.MarkPersistentFlagRequired(FLAG_DST)
	v.BindPFlag(FLAG_DST, c.PersistentFlags().Lookup(FLAG_DST))
}

func FlagsAddServiceFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_SERVICE, "", "service")
	c.MarkPersistentFlagRequired(FLAG_SERVICE)
	v.BindPFlag(FLAG_SERVICE, c.PersistentFlags().Lookup(FLAG_SERVICE))
}

func FlagsAddRestoreArchivesDir(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().String(FLAG_RESTORE_ARCHIVE, "/tmp/jerrie/restore", "restore archive")
	c.MarkPersistentFlagRequired(FLAG_RESTORE_ARCHIVE)
	v.BindPFlag(FLAG_RESTORE_ARCHIVE, c.PersistentFlags().Lookup(FLAG_RESTORE_ARCHIVE))
}

func KubeClientGet(v *viper.Viper) (*kube.Client, error) {
	// use the current context in kubeconfig
	kubeMasterURL := v.GetString(FLAG_KUBE_MASTER_URL)
	kubeConfigPath := v.GetString(FLAG_KUBE_CONFIG_PATH)
	return kube.NewClient(kubeMasterURL, kubeConfigPath)
}

func KubeConfGet(v *viper.Viper) (*restclient.Config, error) {
	// use the current context in kubeconfig
	kubeConfigPath := v.GetString(FLAG_KUBE_CONFIG_PATH)
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("could not read kube config from %s: %w", kubeConfigPath, err)
	}
	return kubeConfig, nil
}

func CMDDBRun(v *viper.Viper) *core.DBBadger {
	dbDir := v.GetString(FLAG_DB_DIR)

	core.Log.Warnf("opening database at %s", dbDir)
	opts := badger.DefaultOptions(dbDir)
	opts = opts.WithLogger(core.Log)
	opts = opts.WithSyncWrites(false)
	opts = opts.WithValueLogLoadingMode(options.FileIO)
	opts = opts.WithTableLoadingMode(options.FileIO)
	opts = opts.WithNumVersionsToKeep(0)
	dbBadger := core.NewDBBadger(&opts, core.Log)
	return dbBadger
}
