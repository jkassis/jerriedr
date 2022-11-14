package main

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	v := viper.New()

	c := &cobra.Command{
		Use:   "servicesnapshotget",
		Short: "Retrieve snapshots from services and store in a local folder.",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDServiceSnapshotGet(v)
		},
	}

	// kube
	CMDKubeConfig(c, v)

	// services
	c.PersistentFlags().StringP(FLAG_SERVICE, "s", "", "a <host>:<port> that responds to requests at '<host>:<port>/<version>/backup' by placing backup files in /var/data/single/<host>-<port>-server-0/backup/<timestamp>.bak")
	v.BindPFlag(FLAG_SERVICE, c.PersistentFlags().Lookup(FLAG_SERVICE))

	MAIN.AddCommand(c)
}

func CMDServiceSnapshotGet(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("ServiceSnapshotGet: starting")

	// for each service
	serviceSpecs := v.GetStringSlice(FLAG_SERVICE)
	ServiceSnapshotGet(v, serviceSpecs)

	duration := time.Since(start)
	core.Log.Warnf("ServiceSnapshotGet: took %s", duration.String())
}

func ServiceSnapshotGet(v *viper.Viper, serviceSpecs []string) {
	if len(serviceSpecs) == 0 {
		core.Log.Fatalf("No services specified.")
	}

	// get http protocol and backup protocol version
	protocol := v.GetString(FLAG_PROTOCOL)
	core.Log.Infof("got protocol: %s", protocol)
	version := v.GetString(FLAG_VERSION)
	core.Log.Infof("got version: %s", version)

	// parse service specs
	hostServices, podServices, err := parseServiceSpecs(serviceSpecs)
	if err != nil {
		core.Log.Fatalf("ServiceSnapshotGet: %v", err)
	}

	// establish a WaitGroup
	wg := sync.WaitGroup{}

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)

	// start backups on podServices
	for _, service := range podServices {
		wg.Add(1)
		go func(service *PodService) {
			defer wg.Done()

			// yes. make sure we have a kube client
			if kubeErr != nil {
				core.Log.Fatalf("kube client initialization failed: %v", kubeErr)
			}

			// get the pod
			pod, err := kubeClient.GetPod(service.PodName, service.PodNamespace)
			if err != nil {
				core.Log.Fatalf("could not get pod: %v", err)
			}

			// run ls
			output, err := kubeClient.ExecSync(pod, "", []string{"ls", service.Dir + "/backup"}, nil)
			if err != nil {
				core.Log.Errorf("could not execute ls on %s", service.Spec)
				return
			}

			// check for files
			files := strings.Split(output, "\n")
			if len(files) == 0 {
				core.Log.Errorf("no backups found in %s", service.Dir+"/backup")
			}

			// sort files
			sort.Strings(files)
			sort.Sort(sort.Reverse(sort.StringSlice(files)))

			// the first one is the most recent!
			latestBackup := files[0]

			// copy it
			fileFullPath := service.Dir + "/backup/" + latestBackup
			err = kubeClient.CopyFromPod(&kube.FileSpec{
				PodNamespace: service.PodNamespace,
				PodName:      service.PodName,
				File:         fileFullPath,
			}, &kube.FileSpec{
				PodNamespace: "",
				PodName:      "",
				File:         "/tmp/" + latestBackup,
			}, pod, "")

			if err != nil {
				core.Log.Errorf("error getting file %s: %v", fileFullPath, err)
				return
			}

			// warn
			core.Log.Warnf("files: %v", files)
		}(service)
	}

	// start backups on hostServices
	for _, service := range hostServices {
		wg.Add(1)
		go func(service *HostService) {
			defer wg.Done()

		}(service)
	}

	wg.Wait()
}
