package schema

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/http"
	"github.com/jkassis/jerriedr/cmd/kube"
	"golang.org/x/sync/errgroup"
)

func ServiceNew() *Service {
	service := &Service{}
	return service
}

type Service struct {
	BackupURL     string
	Host          string
	KubeContainer string
	KubeName      string
	KubeNamespace string
	Name          string
	Port          int
	RestoreURL    string
	RestorePath   string
	Scheme        string
	Spec          string
}

func (s *Service) Parse(spec string) error {
	parts := strings.Split(spec, "|")
	s.Scheme = parts[0]

	if s.Scheme == "statefulset" {
		err := fmt.Errorf(
			"%s must be statefulset|<kubeNamespace>/<kubeName>(/<container>)?|port|"+
				"<backupURL>|<restoreURL>|<restoreDirPath>", spec)

		if len(parts) != 6 {
			return err
		}

		{
			statefulSet := parts[1]
			if !strings.Contains(statefulSet, "/") {
				return err
			}

			statefulSetParts := strings.Split(statefulSet, "/")
			s.KubeNamespace = statefulSetParts[0]
			if s.KubeNamespace == "" {
				return err
			}

			s.KubeName = statefulSetParts[1]
			if s.KubeName == "" {
				return err
			}
			s.Name = s.KubeName

			if len(statefulSetParts) == 3 {
				s.KubeContainer = statefulSetParts[2]
			}
		}

		{
			port, converr := strconv.Atoi(parts[2])
			if converr != nil {
				return err
			}
			s.Port = port
		}

		s.BackupURL = parts[3]
		if s.BackupURL == "" {
			return err
		}

		s.RestoreURL = parts[4]
		if s.RestoreURL == "" {
			return err
		}

		s.RestorePath = parts[5]
		if s.RestorePath == "" {
			return err
		}
	} else if s.Scheme == "pod" {
		err := fmt.Errorf(
			"%s must be pod|<service>|<kubeNamespace>/<kubeName>(/<container>)?|"+
				"<path>|<backupURL>|<restoreURL>|<restoreDirPath>", spec)

		if len(parts) != 7 {
			return err
		}

		s.Name = parts[1]
		if s.Name == "" {
			return err
		}

		{
			pod := parts[2]
			if !strings.Contains(pod, "/") {
				return err
			}

			{
				podParts := strings.Split(pod, "/")
				s.KubeNamespace = podParts[0]
				if s.KubeNamespace == "" {
					return err
				}

				s.KubeName = podParts[1]
				if s.KubeName == "" {
					return err
				}

				if len(podParts) == 3 {
					s.KubeContainer = podParts[2]
				}
			}
		}

		{
			port, converr := strconv.Atoi(parts[3])
			if converr != nil {
				return err
			}
			s.Port = port
		}

		s.BackupURL = parts[4]
		if s.BackupURL == "" {
			return err
		}

		s.RestoreURL = parts[5]
		if s.RestoreURL == "" {
			return err
		}

		s.RestorePath = parts[6]
		if s.RestorePath == "" {
			return err
		}
	} else if s.Scheme == "host" {
		err := fmt.Errorf("%s must be host|<service>|<hostName>|<port>|"+
			"<backupURL>|<restoreURL>|<restorePath>", spec)

		if len(parts) != 7 {
			return err
		}

		s.Name = parts[1]
		if s.Name == "" {
			return err
		}

		{
			s.Host = parts[2]
			if s.Host == "" {
				return err
			}
		}

		{
			port, converr := strconv.Atoi(parts[3])
			if converr != nil {
				return err
			}
			s.Port = port
		}

		s.BackupURL = parts[4]
		if s.BackupURL == "" {
			return err
		}

		s.RestoreURL = parts[5]
		if s.RestoreURL == "" {
			return err
		}

		s.RestorePath = parts[6]
		if s.RestorePath == "" {
			return err
		}
	} else if s.Scheme == "local" {
		err := fmt.Errorf("%s must be local|<service>|<port>|<backupURL>|"+
			"<restoreURL>|<restorePath>", spec)

		if len(parts) != 6 {
			return err
		}

		s.Host = "localhost"

		s.Name = parts[1]
		if s.Name == "" {
			return err
		}

		{
			port, convErr := strconv.Atoi(parts[2])
			if convErr != nil {
				return err
			}
			s.Port = port
		}

		s.BackupURL = parts[3]
		if s.BackupURL == "" {
			return err
		}
		s.RestoreURL = parts[4]
		if s.RestoreURL == "" {
			return err
		}
		s.RestorePath = parts[5]
		if s.RestorePath == "" {
			return err
		}
	} else {
		return fmt.Errorf(
			"%s must be <scheme>|<schemeSpec> where <scheme> => statefulset | pod | "+
				"host | local: %s", spec, s.Scheme)
	}

	s.Spec = spec
	return nil
}

func (s *Service) IsPod() bool {
	return s.Scheme == "pod"
}

func (s *Service) IsHost() bool {
	return s.Scheme == "host"
}

func (s *Service) IsLocal() bool {
	return s.Scheme == "local"
}

func (s *Service) IsStatefulSet() bool {
	return s.Scheme == "statefulset"
}

func (s *Service) Replicas(kubeClient *kube.KubeClient) (n int, err error) {
	if !s.IsStatefulSet() {
		return 0, fmt.Errorf("iterating requires a statefulset")
	}

	if kubeClient == nil {
		return 0, fmt.Errorf("must have kubeClient")
	}

	// get the statefulSet
	statefulSet, err := kubeClient.StatefulSetGetByName(s.KubeNamespace, s.KubeName)
	if err != nil {
		return 0, fmt.Errorf("could not get statefulset %s: %w", s.KubeName, err)
	}

	// for each replica...
	replicas := statefulSet.Spec.Replicas
	return int(*replicas), nil
}

func (s *Service) ServicePodGet(replica int) (*Service, error) {
	if !s.IsStatefulSet() {
		return nil, fmt.Errorf("%s is not a statefulset spec", s.Spec)
	}
	podName := s.KubeName + "-" + strconv.Itoa(replica)

	// TODO check this
	host := fmt.Sprintf(
		"%s.%s-int.%s.svc.cluster.local", podName, s.KubeName, s.KubeNamespace)
	return &Service{
		Host:          host,
		KubeContainer: s.KubeContainer,
		KubeName:      podName,
		KubeNamespace: s.KubeNamespace,
		Name:          s.Name,
		Port:          s.Port,
		Scheme:        "pod",
		Spec: fmt.Sprintf("pod|%s|%s/%s/%s|%d|%s|%s",
			s.Name,
			s.KubeNamespace,
			podName,
			s.KubeContainer,
			s.Port,
			s.BackupURL,
			s.RestoreURL),
	}, nil
}

func (s *Service) ForEachServicePod(
	kubeClient *kube.KubeClient, fn func(podService *Service) error) (err error) {
	replicas, err := s.Replicas(kubeClient)
	if err != nil {
		return err
	}

	eg := errgroup.Group{}
	for i := 0; i < replicas; i++ {
		podService, err := s.ServicePodGet(i)
		if err != nil {
			return err
		}
		eg.Go(func() error {
			return fn(podService)
		})
	}
	return eg.Wait()
}

// Snap initiates a snapshop / backup of the service.
// the snap message is posted to the raft, so there is no
// need to send this to each server in the StatefulSet
func (s *Service) Snap(kubeClient *kube.KubeClient) (err error) {
	core.Log.Warnf("running remote backup for %s", s.Spec)

	var reqURL string
	{
		if s.IsStatefulSet() {
			b, err := s.ServicePodGet(0)
			if err != nil {
				return err
			}
			return b.Snap(kubeClient)
		} else if s.IsPod() {
			// yes. make sure we have a kube client
			if kubeClient != nil {
				return fmt.Errorf("need kubeClient")
			}

			// forward a local port
			forwardedPort, err := kubeClient.PortForward(&kube.PortForwardRequest{
				LocalPort:    0,
				PodName:      s.KubeName,
				PodNamespace: s.KubeNamespace,
				PodPort:      s.Port,
			})
			if err != nil {
				return fmt.Errorf(
					"could not port forward to kube service %s: %v", s.Spec, err)
			}
			localPort := forwardedPort.Local
			reqURL = fmt.Sprintf(
				"%s://%s:%d/raft/leader/read", "http", "localhost", localPort)
		} else if s.IsHost() {
			reqURL = fmt.Sprintf("http://%s:%d/raft/leader/read", s.Host, s.Port)
		} else if s.IsLocal() {
			reqURL = fmt.Sprintf("http://localhost:%d/raft/leader/read", s.Port)
		}
	}

	// make the request
	reqBody := fmt.Sprintf(
		`{ "UUID": "%s", "Fn": "%s", "Body": {} }`, uuid.NewString(), s.BackupURL)
	if res, err := http.Post(reqURL, "application/json", reqBody); err != nil {
		return fmt.Errorf("could not request %s: %v", reqURL, err)
	} else {
		core.Log.Warnf("finished %s: %s", s.KubeName, res)
	}
	return nil
}

// Reset calls the reset endpoint for the service.
// The service defines the behavior, but this should basically clean
// the datasource in preparation for data loading.
func (s *Service) Reset(kubeClient *kube.KubeClient) (err error) {
	if s.IsStatefulSet() {
		return s.ForEachServicePod(kubeClient, func(servicePod *Service) error {
			return servicePod.Reset(kubeClient)
		})
	}

	// make the HTTP request to the reset endpoint
	reqURL := fmt.Sprintf("http://%s:%d/v1/Reset/App", s.Host, s.Port)
	core.Log.Warnf("trying: %s", reqURL)
	reqBody := fmt.Sprintf(
		`{ "UUID": "%s", "Fn": "/v1/Reset/App", "Body": {} }`, uuid.NewString())
	if res, err := http.Post(reqURL, "application/json", reqBody); err != nil {
		return fmt.Errorf("%s: %s: %v", reqURL, res, err)
	} else {
		core.Log.Warnf("%s: %s", reqURL, res)
	}

	return nil
}

// Stage prepares a service for restoration. We might stage and restore
// multiple data files to the service (eg. when we restore prod data to a
// dev service), so we break this out.
func (s *Service) Stage(
	kubeClient *kube.KubeClient, srcArchiveFile *ArchiveFile) error {
	if s.IsStatefulSet() {
		if s.IsStatefulSet() {
			return s.ForEachServicePod(kubeClient, func(servicePod *Service) error {
				return servicePod.Reset(kubeClient)
			})
		}
	}

	if s.IsPod() {
		pod, err := kubeClient.PodGetByName(s.KubeNamespace, s.KubeName)
		if err != nil {
			return err
		}

		// reset the restore folder
		_, err = kubeClient.Rm(s.RestorePath, pod, "")
		if err != nil {
			return err
		}

		// make the restore folder
		_, err = kubeClient.MkDir(s.RestorePath, pod, "")
		if err != nil {
			return err
		}

		// can only stage archives on the same machine
		if srcArchiveFile.Archive.KubeName != s.KubeName ||
			srcArchiveFile.Archive.KubeNamespace != s.KubeNamespace {
			return fmt.Errorf("can only stage files on the same "+
				"pod. trying to stage %s on %s",
				srcArchiveFile.Archive.KubeName,
				s.KubeName)
		}

		// make a symlink
		srcArchiveFilePath := srcArchiveFile.Archive.Path + "/" + srcArchiveFile.Name
		dstArchiveFilePath := s.RestorePath + "/" + srcArchiveFile.Name
		_, err = kubeClient.Ln(s.RestorePath, dstArchiveFilePath, pod, "")
		if err != nil {
			return fmt.Errorf("cound not create symlink: src %s to %s: %v",
				srcArchiveFilePath, dstArchiveFilePath, err)
		}
	} else if s.IsHost() {
		return fmt.Errorf("cannot Stage for hosts yet")
	} else if s.IsLocal() {
		// clear the content of the restore folder
		if err := os.RemoveAll(s.RestorePath); err != nil {
			return fmt.Errorf("cound not clear the content of the restore folder: %v", err)
		}

		// recreate it
		if err := os.MkdirAll(s.RestorePath, 0774); err != nil {
			return fmt.Errorf("cound not create the restore folder: %v", err)
		}

		// can only do local to local
		if srcArchiveFile.Archive.Scheme != "local" {
			return fmt.Errorf("can only restore to local from local. srcArchive is %v",
				srcArchiveFile.Archive)
		}

		// make a symlink
		srcArchiveFilePath := srcArchiveFile.Archive.Path + "/" + srcArchiveFile.Name
		dstArchiveFilePath := s.RestorePath + "/" + srcArchiveFile.Name
		err := os.Symlink(srcArchiveFilePath, dstArchiveFilePath)
		if err != nil {
			return fmt.Errorf("cound not create symlink: src %s to %s: %v",
				srcArchiveFilePath, dstArchiveFilePath, err)
		}
	}
	return nil
}

// Restore actuates the actual loading of data after staging
func (s *Service) Restore(kubeClient *kube.KubeClient) error {
	if s.IsStatefulSet() {
		return s.ForEachServicePod(kubeClient, func(servicePod *Service) error {
			return servicePod.Restore(kubeClient)
		})
	}

	core.Log.Warnf("restoring %s", s.Name)
	reqURL := fmt.Sprintf("http://%s:%d%s", s.Host, s.Port, s.RestoreURL)
	core.Log.Warnf("trying: %s", reqURL)
	reqBod := fmt.Sprintf(`{ "UUID": "%s", "Fn": "/v1/Restore", "Body": {} }`,
		uuid.NewString())
	if res, err := http.Post(reqURL, "application/json", reqBod); err != nil {
		return fmt.Errorf("%s: %s: %v", reqURL, res, err)
	} else {
		core.Log.Warnf("%s: %s", reqURL, res)
	}

	return nil
}

// RAFTReset resets the raft after a restore. This is necessary in
// The service decides how to do this, ultimately.
func (s *Service) RAFTReset(kubeClient *kube.KubeClient) error {
	if s.IsStatefulSet() {
		return s.ForEachServicePod(kubeClient, func(servicePod *Service) error {
			return servicePod.RAFTReset(kubeClient)
		})
	}

	reqURL := fmt.Sprintf("http://%s:%d/v1/Reset/Raft", s.Host, s.Port)
	reqBod := fmt.Sprintf(`{ "UUID": "%s", "Fn": "/v1/Reset/Raft", "Body": {} }`,
		uuid.NewString())
	if res, err := http.Post(reqURL, "application/json", reqBod); err != nil {
		return fmt.Errorf("%s: %s: %v", reqURL, res, err)
	} else {
		core.Log.Warnf("%s: %s", reqURL, res)
	}
	return nil
}
