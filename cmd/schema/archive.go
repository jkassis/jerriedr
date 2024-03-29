package schema

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"golang.org/x/sync/errgroup"
)

const FLAG_ARCHIVE = "archive"

func ArchiveNew() *Archive {
	archive := &Archive{}
	archive.Files = make([]*ArchiveFile, 0)
	archive.Filters = make([]*TimeFilter, 0)
	return archive
}

type Archive struct {
	Files         []*ArchiveFile
	FilesFiltered []*ArchiveFile
	Filters       []*TimeFilter
	Host          string
	KubeContainer string
	KubeName      string
	KubeNamespace string
	Path          string
	Parent        *Archive
	Scheme        string
	ServiceName   string
	Spec          string
}

func (a *Archive) Parse(spec string) error {
	parts := strings.Split(spec, "|")
	a.Scheme = parts[0]
	if a.Scheme == "statefulset" {
		err := fmt.Errorf("%s must be statefulset|<namespace>/<statefulSet>|<pathPattern> where <pathPattern> can contain '<pod>' to insert the pod Name", spec)

		if len(parts) != 3 {
			return err
		}

		{
			serviceSpec := parts[1]
			if !strings.Contains(serviceSpec, "/") {
				return err
			}

			serviceSpecParts := strings.Split(serviceSpec, "/")
			a.KubeNamespace = serviceSpecParts[0]
			if a.KubeNamespace == "" {
				return err
			}

			a.KubeName = serviceSpecParts[1]
			if a.KubeName == "" {
				return err
			}
			a.ServiceName = a.KubeName
		}

		a.Path = parts[2]
		if a.Path == "" {
			return err
		}
	} else if a.Scheme == "pod" {
		err := fmt.Errorf("%s must be pod|<namespace>/<statefulSet>/<pod>|<path>", spec)

		if len(parts) != 3 {
			return err
		}

		{
			serviceSpec := parts[1]
			if !strings.Contains(serviceSpec, "/") {
				return err
			}

			serviceSpecParts := strings.Split(serviceSpec, "/")
			a.KubeNamespace = serviceSpecParts[0]
			if a.KubeNamespace == "" {
				return err
			}

			a.ServiceName = serviceSpecParts[1]
			if a.ServiceName == "" {
				return err
			}

			a.KubeName = serviceSpecParts[2]
			if a.KubeName == "" {
				return err
			}
		}

		a.Path = parts[2]
		if a.Path == "" {
			return err
		}
	} else if a.Scheme == "host" {
		err := fmt.Errorf("%s must be host|<hostName>/<service>|<path>", spec)

		if len(parts) != 3 {
			return err
		}

		{
			serviceSpec := parts[1]
			if !strings.Contains(serviceSpec, "/") {
				return err
			}

			serviceSpecParts := strings.Split(parts[1], "/")
			a.Host = serviceSpecParts[0]
			if a.Host == "" {
				return err
			}

			a.ServiceName = serviceSpecParts[1]
			if a.Host == "" {
				return err
			}
		}

		a.Path = parts[2]
		if a.Path == "" {
			return err
		}
	} else if a.Scheme == "local" {
		err := fmt.Errorf("%s must be local|<service>|<path>", spec)

		if len(parts) != 3 {
			return err
		}

		a.ServiceName = parts[1]
		if a.ServiceName == "" {
			return err
		}

		a.Path = parts[2]
		if a.Path == "" {
			return err
		}
	} else {
		return fmt.Errorf("%s must be <scheme>|<schemeSpec> where <scheme> => statefulset | pod | host | local: %s", spec, a.Scheme)
	}
	a.Spec = spec
	return nil
}

func (a *Archive) IsPod() bool {
	return a.Scheme == "pod"
}

func (a *Archive) IsHost() bool {
	return a.Scheme == "host"
}

func (a *Archive) IsLocal() bool {
	return a.Scheme == "local"
}

func (a *Archive) IsStatefulSet() bool {
	return a.Scheme == "statefulset"
}

func (a *Archive) PodArchiveGet(replica int) (*Archive, error) {
	if !a.IsStatefulSet() {
		return nil, fmt.Errorf("%s is not a statefulset spec", a.Spec)
	}
	podName := a.KubeName + "-" + strconv.Itoa(replica)
	podPath := strings.ReplaceAll(a.Path, "<pod>", podName)
	return &Archive{
		Files:         make([]*ArchiveFile, 0),
		Host:          "",
		KubeContainer: a.KubeContainer,
		KubeName:      podName,
		KubeNamespace: a.KubeNamespace,
		Path:          podPath,
		Parent:        a,
		Scheme:        "pod",
		ServiceName:   a.ServiceName,
		Spec:          fmt.Sprintf("pod|%s/%s/%s|%s", a.KubeNamespace, a.ServiceName, podName, podPath),
	}, nil
}

func (a *Archive) Replicas(kubeClient *kube.Client) (n int, err error) {
	if !a.IsStatefulSet() {
		return 0, fmt.Errorf("iterating requires a statefulset")
	}

	if kubeClient == nil {
		return 0, fmt.Errorf("must have kubeClient")
	}

	// get the statefulSet
	statefulSet, err := kubeClient.StatefulSetGetByName(a.KubeNamespace, a.KubeName)
	if err != nil {
		return 0, fmt.Errorf("could not get statefulset %s: %w", a.KubeName, err)
	}

	// for each replica...
	replicas := statefulSet.Spec.Replicas
	return int(*replicas), nil
}

func (a *Archive) FilesFetch(kubeClient *kube.Client) error {
	files := make([]*ArchiveFile, 0)

	if a.IsStatefulSet() {
		// for each replica...
		eg := errgroup.Group{}
		replicas, err := a.Replicas(kubeClient)
		if err != nil {
			return err
		}
		for i := 0; i < replicas; i++ {
			i := i
			eg.Go(func() error {
				podArchive, err := a.PodArchiveGet(i)
				if err != nil {
					return fmt.Errorf("could not get podArchiveSpec from statefulSetArchiveSpec: %v", err)
				}

				err = podArchive.FilesFetch(kubeClient)
				if err != nil {
					return fmt.Errorf("could not get files for %s of %s: %v", podArchive.Spec, a.Spec, err)
				}
				files = append(files, podArchive.Files...)
				return nil
			})
		}
		if err = eg.Wait(); err != nil {
			return err
		}
	} else if a.IsPod() {
		if kubeClient == nil {
			return fmt.Errorf("must have kubeClient")
		}

		// get the pod
		pod, err := kubeClient.PodGetByName(a.KubeNamespace, a.KubeName)
		if err != nil {
			return fmt.Errorf("could not get pod: %v", err)
		}

		core.Log.Warnf("fetching file list for pod archive %s", a.Spec)
		podFileNames, err := kubeClient.Ls(a.Path, pod, a.KubeContainer)
		if err != nil {
			return fmt.Errorf("could not list files for podSpec %s: %v", a.Spec, err)
		}

		for _, podFileName := range podFileNames {
			file := &ArchiveFile{
				Archive: a,
				Name:    podFileName,
			}
			err := file.TimestampParseFromName()
			if err == nil {
				files = append(files, file)
				core.Log.Debugf("found %s/%s", a.Spec, podFileName)
			} else {
				core.Log.Warnf("could not parse archiveFile timestamp from %s", podFileName)
			}
		}
	} else if a.IsLocal() {
		dirEntries, err := os.ReadDir(a.Path)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range dirEntries {
			if file.IsDir() {
				continue
			}
			fileName := &ArchiveFile{
				Archive: a,
				Name:    file.Name(),
			}
			err := fileName.TimestampParseFromName()
			if err == nil {
				files = append(files, fileName)
				core.Log.Debugf("found %s/%s", a.Spec, fileName)
			} else {
				core.Log.Warnf("could not parse archiveFile timestamp from %s", fileName)
			}
		}
	}

	sort.Sort(ByMostRecent(files))
	a.Files = files
	a.FilesFiltered = files
	return nil
}

func (a *Archive) FilterAdd(tf *TimeFilter) {
	a.Filters = append(a.Filters, tf)
}

func (a *Archive) FiltersClear() {
	a.Filters = make([]*TimeFilter, 0)
}

func (a *Archive) Filter() {
	filteredArchiveFiles := make([]*ArchiveFile, 0)

	for _, file := range a.Files {
		for _, filter := range a.Filters {
			if !file.FilterIsOK(filter) {
				continue
			}
		}
		filteredArchiveFiles = append(filteredArchiveFiles, file)
	}

	a.FilesFiltered = filteredArchiveFiles
}

func (ss *Archive) FileGetFilteredBefore(t time.Time) *ArchiveFile {
	for _, file := range ss.FilesFiltered {
		if file.Time.Before(t) {
			return file
		}
	}

	return nil
}

type BySpec []*Archive

func (afs BySpec) Len() int {
	return len(afs)
}

func (afs BySpec) Less(i, j int) bool {
	return afs[i].Spec < afs[j].Spec
}

func (afs BySpec) Swap(i, j int) {
	t := afs[i]
	afs[i] = afs[j]
	afs[j] = t
}
