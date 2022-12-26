package schema

import (
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"golang.org/x/sync/errgroup"
)

func EnvRestore(kubeClient *kube.Client, srcArchiveSpecs, dstServiceSpecs []string) {
	var err error

	// get src and dst archiveSets and serviceSets from specs
	var srcArchiveSet *ArchiveSet
	{
		srcArchiveSet = ArchiveSetNew()
		err := srcArchiveSet.ArchiveAddAll(srcArchiveSpecs, "")
		if err != nil {
			core.Log.Fatalf("could not add srcArchive %v", err)
		}
	}

	//
	var dstServiceSet *ServiceSet
	{
		dstServiceSet = ServiceSetNew()
		err = dstServiceSet.ServiceAddAll(dstServiceSpecs)
		if err != nil {
			core.Log.Fatalf("could not add dstArchive %v", err)
		}
	}

	// User picks the snapshot
	srcArchiveFileSet, err := srcArchiveSet.PickSnapshot()
	if err != nil {
		core.Log.Fatalf("snapshot not picked... cancelling operation")
	}

	// get all dstServices
	dstServices := map[string]*Service{}
	{
		for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
			dstService, err := dstServiceSet.ServiceGetByServiceName(srcArchiveFile.Archive.ServiceName)
			if err != nil {
				core.Log.Fatalf("could not find dstService to match srcArchiveFile '%s': %v", srcArchiveFile.Name, err)
			}
			dstServices[dstService.Name] = dstService
		}
	}

	eg := errgroup.Group{}
	for _, dstService := range dstServices {
		dstService := dstService
		eg.Go(func() (err error) {
			if err = dstService.StartStop(kubeClient, false); err != nil {
				return err
			}
			if err = dstService.WaitForDrain(kubeClient); err != nil {
				return err
			}
			return dstService.Reset(kubeClient)
		})
	}
	if err = eg.Wait(); err != nil {
		core.Log.Fatal(err)
	}

	// for each archiveFile
	for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
		// have to do these one at a time.
		dstService := dstServices[srcArchiveFile.Archive.ServiceName]

		if err = dstService.Stage(kubeClient, srcArchiveFile); err != nil {
			core.Log.Fatalf("could not stage %s to %s: %v", srcArchiveFile.Name, dstService.Spec, err)
		}

		if err = dstService.Restore(kubeClient); err != nil {
			core.Log.Fatalf("could not stage %s to %s: %v", srcArchiveFile.Name, dstService.Spec, err)
		}
	}

	// Reset RAFTs
	eg = errgroup.Group{}
	for _, dstService := range dstServices {
		dstService := dstService
		eg.Go(func() (err error) {
			if err = dstService.RAFTReset(kubeClient); err != nil {
				return err
			}
			return dstService.StartStop(kubeClient, true)
		})
	}
	if err = eg.Wait(); err != nil {
		core.Log.Fatal(err)
	}
}
