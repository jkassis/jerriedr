package schema

import (
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
)

func EnvRestore(kubeClient *kube.Client, srcArchiveSpecs, dstServiceSpecs []string) {
	var err error

	// get srcArchiveSet from specs
	var srcArchiveSet *ArchiveSet
	{
		srcArchiveSet = ArchiveSetNew()
		err := srcArchiveSet.ArchiveAddAll(srcArchiveSpecs, "")
		if err != nil {
			core.Log.Fatalf("could not add srcArchive %v", err)
		}
	}

	// get dstServiceSet from specs
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

	// narrow down the dstServiceSet to those with references in the srcArchiveFileSet
	{
		newDstServiceSet := ServiceSetNew()
		for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
			dstService, err := dstServiceSet.ServiceGetByName(srcArchiveFile.Archive.ServiceName)
			if err != nil {
				core.Log.Fatalf("could not find dstService to match srcArchiveFile '%s': %v", srcArchiveFile.Name, err)
			}
			newDstServiceSet.ServiceAdd(dstService)
		}
		dstServiceSet = newDstServiceSet
	}

	// Prepare all endpoints
	if err = dstServiceSet.DoOncePerEndpoint(
		func(dstService *Service) (err error) {
			if err = dstService.StartStop(kubeClient, false); err != nil {
				return err
			}
			if err = dstService.WaitForDrain(kubeClient); err != nil {
				return err
			}
			return dstService.Reset(kubeClient)
		}); err != nil {
		core.Log.Fatal(err)
	}

	// for each archiveFile
	for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
		dstService, _ := dstServiceSet.ServiceGetByName(srcArchiveFile.Archive.ServiceName)

		// do this one at a time.
		if err = dstService.Stage(kubeClient, srcArchiveFile); err != nil {
			core.Log.Fatalf("could not stage %s to %s: %v", srcArchiveFile.Name, dstService.Spec, err)
		}

		if err = dstService.Restore(kubeClient); err != nil {
			core.Log.Fatalf("could not restore %s to %s: %v", srcArchiveFile.Name, dstService.Spec, err)
		}
	}

	// Restart all endpoints
	if err = dstServiceSet.DoOncePerEndpoint(
		func(dstService *Service) (err error) {
			if err = dstService.RAFTReset(kubeClient); err != nil {
				return err
			}
			return dstService.StartStop(kubeClient, true)
		}); err != nil {
		core.Log.Fatal(err)
	}
}
