package kube

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/jkassis/jerrie/core"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

// Client is a kube client
type Client struct {
	Clientset      *kubernetes.Clientset
	Config         *rest.Config
	KubeConfigPath string
	MasterURL      string
	Rand           *rand.Rand
}

// NewClient returns a new, init'd kube client
func NewClient(masterURL, kubeConfigPath string) (*Client, error) {
	kubeClient := &Client{
		KubeConfigPath: kubeConfigPath,
		MasterURL:      masterURL,
	}
	err := kubeClient.Init()
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

// Init starts the kubernetes client
func (c *Client) Init() error {
	config, err := clientcmd.BuildConfigFromFlags(c.MasterURL, c.KubeConfigPath)
	if err != nil {
		return fmt.Errorf("could not load kube config from %s: %v", c.KubeConfigPath, err)
	}

	c.Config = config
	gv := v1.SchemeGroupVersion
	c.Config.GroupVersion = &gv
	// config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	// create the clientset
	clientset, err := kubernetes.NewForConfig(c.Config)
	if err != nil {
		return err
	}

	c.Clientset = clientset

	c.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return nil
}

// DeploymentGetByNamespace returns deployments
func (c *Client) DeploymentGetByNamespace(namespace string, pattern string) ([]string, error) {
	matchingDeploymentNames := []string{}
	deploymentsList, err := c.Clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, deployment := range deploymentsList.Items {
		matched, err := regexp.MatchString(pattern, deployment.GetName())
		if err != nil {
			return nil, err
		}
		if matched {
			matchingDeploymentNames = append(matchingDeploymentNames, deployment.Name)
		}
	}

	sort.Slice(matchingDeploymentNames, func(i, j int) bool {
		return matchingDeploymentNames[i] > matchingDeploymentNames[j]
	})
	return matchingDeploymentNames, nil
}

// StatefulSetGetByNamespace returns deployments
func (c *Client) StatefulSetGetByNamespace(namespace string, pattern string) ([]string, error) {
	matchingDeploymentNames := []string{}
	statefulSetList, err := c.Clientset.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, statefulSet := range statefulSetList.Items {
		matched, err := regexp.MatchString(pattern, statefulSet.GetName())
		if err != nil {
			return nil, err
		}
		if matched {
			matchingDeploymentNames = append(matchingDeploymentNames, statefulSet.Name)
		}
	}

	sort.Slice(matchingDeploymentNames, func(i, j int) bool {
		return matchingDeploymentNames[i] > matchingDeploymentNames[j]
	})
	return matchingDeploymentNames, nil
}

// StatefulSetGetByName returns deployments
func (c *Client) StatefulSetGetByName(namespace string, name string) (*v1.StatefulSet, error) {
	statefulSet, err := c.Clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		return nil, fmt.Errorf("pod %s in namespace %s not found", statefulSet, namespace)
	} else if statusError, isStatus := err.(*k8sErrors.StatusError); isStatus {
		return nil, fmt.Errorf("error getting pod %s in namespace %s: %v", name, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		return nil, err
	}
	return statefulSet, nil
}

// PodGetByName returns a pod
func (c *Client) ServiceGetByName(
	namespace,
	name string) (*corev1.Service, error) {
	service, err := c.Clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		return nil, fmt.Errorf("pod %s in namespace %s not found", service, namespace)
	} else if statusError, isStatus := err.(*k8sErrors.StatusError); isStatus {
		return nil, fmt.Errorf("error getting pod %s in namespace %s: %v", name, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		return nil, err
	}
	return service, nil
}

// PodGetByName returns a pod
func (c *Client) PodGetByName(
	namespace,
	name string) (*corev1.Pod, error) {
	pod, err := c.Clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		return nil, fmt.Errorf("pod %s in namespace %s not found", pod, namespace)
	} else if statusError, isStatus := err.(*k8sErrors.StatusError); isStatus {
		return nil, fmt.Errorf("error getting pod %s in namespace %s: %v", name, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		return nil, err
	}
	return pod, nil
}

// PodGetByNamespace returns all pods in a cluster
func (c *Client) PodGetByNamespace(namespace string) (*corev1.PodList, error) {
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// PodGetByDeploymentName returns all pods for a deployment
func (c *Client) PodGetByDeploymentName(namespace, deploymentName string) (*corev1.PodList, error) {
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "deployment=" + deploymentName,
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// PodGetRandomByDeploymentName return a random pod in the namespace / deployment
func (c *Client) PodGetRandomByDeploymentName(namespace, deploymentName string) (*corev1.Pod, error) {
	// Get a deployment to operate on... we might be wrong
	deployments, err := c.DeploymentGetByNamespace(namespace, deploymentName)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Found these remote deployments: %s\n", strings.Join(deployments, " "))
	firstDeployment := deployments[0]

	// PodList
	logrus.Info("Getting pods for " + namespace + "/" + firstDeployment + "\n")
	podList, err := c.PodGetByDeploymentName(namespace, firstDeployment)
	if err != nil {
		return nil, err
	}
	if len(podList.Items) == 0 {
		return nil, errors.New("found no pods")
	}
	logrus.Infof("Got %d pods for %s/%s\n", len(podList.Items), namespace, deploymentName)

	pod := podList.Items[c.Rand.Intn(len(podList.Items))]
	return &pod, nil
}

// Exec executes a command asynchronously on a given pod
// stdin is piped to the remote shell if provided or nothing if nil
// returns the output from stdout and stderr
func (c *Client) Exec(
	pod *corev1.Pod,
	containerName string,
	command []string,
	stdinReader io.Reader,
	stdoutWriter io.Writer) error {
	stderrReader, stderrWriter := io.Pipe()

	request := c.Clientset.CoreV1().RESTClient().
		// Post().
		Get().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		Param("container", containerName).
		VersionedParams(&corev1.PodExecOptions{
			Command: command,
			Stdin:   stdinReader != nil,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.Config, "POST", request.URL())
	if err != nil {
		return err
	}

	eg := errgroup.Group{}
	eg.Go(func() error {
		var err error
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: stderrWriter,
			Tty:    false,
		})
		if err != nil {
			core.Log.Errorf("error while streaming from kube: %v", err)
		}
		err = stderrWriter.Close()
		if err != nil {
			core.Log.Errorf("trouble closing stdout writer after streaming from kube: %v", err)
		}
		return nil
	})

	// read stderr and convert to err
	eg.Go(func() error {
		stderr := bytes.NewBuffer(nil)
		_, err = io.Copy(stderr, stderrReader)
		if err != nil {
			return err
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("kube.Exec: stderr: %v", stderr)
		}
		return nil
	})

	// return stdout, stderr
	return eg.Wait()
}

// ExecSync executes a command synchronously on a given pod
// stdin is piped to the remote shell if provided or nothing if nil
// returns the output from stdout or err as a string and err
func (c *Client) ExecSync(pod *corev1.Pod, containerName string, command []string, stdin io.Reader) (string, error) {
	stdoutReader, stdoutWriter := io.Pipe()
	eg := errgroup.Group{}

	eg.Go(func() error {
		err := c.Exec(pod, containerName, command, stdin, stdoutWriter)
		if err != nil {
			return err
		}
		return stdoutWriter.Close()
	})

	var stdout []byte
	eg.Go(func() (err error) {
		stdout, err = io.ReadAll(stdoutReader)
		return err
	})

	err := eg.Wait()
	return string(stdout), err
}

// Ls returns a list of files on the pod in the given dir
func (c *Client) Ls(dirPath string, pod *corev1.Pod, containerName string) ([]string, error) {
	dirPath = shellescape.Quote(dirPath)
	cmdArr := []string{"/bin/sh", "-c", "ls " + dirPath}
	stdout, err := c.ExecSync(pod, containerName, cmdArr, nil)
	if err != nil {
		return nil, err
	}
	return strings.Split(stdout, "\n"), nil
}

// MkDir copies a file from local dir to remote
func (c *Client) MkDir(dirPath string, pod *corev1.Pod, containerName string) (stdout string, err error) {
	dirPath = shellescape.Quote(dirPath)
	cmdArr := []string{"/bin/sh", "-c", "mkdir -p " + dirPath}
	logrus.Info("making directory in pod : '" + pod.Name + "'")
	return c.ExecSync(pod, containerName, cmdArr, nil)
}

// Ln creates a softlink
func (c *Client) Ln(srcPath, dstPath string, pod *corev1.Pod, containerName string) (stdout string, err error) {
	srcPath = shellescape.Quote(srcPath)
	dstPath = shellescape.Quote(dstPath)
	cmdArr := []string{"/bin/sh", "-c",
		fmt.Sprintf("ln -s %s %s", srcPath, dstPath)}
	logrus.Info("linking %s to %s in pod %s", srcPath, dstPath, pod.Name)
	return c.ExecSync(pod, containerName, cmdArr, nil)
}

// FileWrite copies content at io.Reader to a file on a pod
func (c *Client) FileWrite(src io.Reader, dstPath string, pod *corev1.Pod, containerName string) (err error) {
	dstPath = shellescape.Quote(dstPath)
	// cmdArr := []string{"/bin/sh", "-c", "mkdir -p " + filepath.Dir(dstFile) + " ; cat > " + dstFile}
	cmdArr := []string{"env", "cat", ">", dstPath}
	return c.Exec(pod, containerName, cmdArr, src, io.Discard)
}

// FileRead copies file on a pod to the writer
func (c *Client) FileRead(src string, dst io.Writer, pod *corev1.Pod, containerName string) (err error) {
	src = shellescape.Quote(src)
	fileStats, err := c.Stat(pod, containerName, src)
	if err != nil {
		return fmt.Errorf("could not get stats for %s: %v", src, err)
	}
	srcFileSize := fileStats.Size
	srcMD5, err := c.MD5Sum(pod, containerName, src)
	if err != nil {
		return fmt.Errorf("could not get md5 for %s: %v", src, err)
	}

	hasher := md5.New()

	// network file transfers are messy...
	// we are going to basically do our own rsync protocol here...
	// if the connection fails due to a premature EOF, we retry.
	// otherwise, we fail
	m := int64(0) // count of total bytes transferred
	buf := make([]byte, 16384)

	for {
		pipeR, pipeW := io.Pipe()

		// transfer
		eg := errgroup.Group{}
		eg.Go(func() (err error) {
			cmdArr := []string{"tail", "-c", fmt.Sprintf("+%d", m+1), src}
			err = c.Exec(pod, containerName, cmdArr, nil, pipeW)
			pipeErr := pipeW.Close() // always close the pipe
			if pipeErr != nil {
				return pipeErr
			}
			return err
		})

		// read from the pipe and forward
		eg.Go(func() error {
			for {
				// read
				o, err := pipeR.Read(buf)
				if o > 0 {
					hasher.Write(buf[:o])
					dst.Write(buf[:o])
					m += int64(o)
				}

				if err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
			}
		})

		// wait
		err = eg.Wait()
		if err != nil {
			return err
		}

		// did we get all the bytes?
		if m >= srcFileSize {
			// yes. compare hashes
			readHash := hex.EncodeToString(hasher.Sum(nil))
			if srcMD5 != readHash {
				return fmt.Errorf("hashes do not match")
			}
			return nil // all done
		}
	}
}

type FileStat struct {
	UID  int64
	GID  int64
	Size int64
	Name string
}

func (c *Client) MD5Sum(pod *corev1.Pod, containerName, path string) (hash string, err error) {
	srcFile := shellescape.Quote(path)
	cmdArr := []string{"env", "md5sum", srcFile}
	response, err := c.ExecSync(pod, containerName, cmdArr, nil)
	if err != nil {
		return "", err
	}

	parts := strings.Split(response, " ")
	return parts[0], nil
}

func (c *Client) Stat(pod *corev1.Pod, containerName, path string) (fileState *FileStat, err error) {
	srcFile := shellescape.Quote(path)

	// this appears to be the Alpine Linux variant... :cringe:
	format := "%u %g %s %N"
	cmdArr := []string{"env", "stat", "-c", format, srcFile}

	response, err := c.ExecSync(pod, containerName, cmdArr, nil)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(response, " ")
	uid, err := strconv.ParseInt(parts[0], 0, 64)
	if err != nil {
		return nil, fmt.Errorf("FileStatGet: could not convert uid for %s got %s", path, response)
	}
	gid, err := strconv.ParseInt(parts[1], 0, 64)
	if err != nil {
		return nil, fmt.Errorf("FileStatGet: could not convert gid for %s got %s", path, response)
	}
	size, err := strconv.ParseInt(parts[2], 0, 64)
	if err != nil {
		return nil, fmt.Errorf("FileStatGet: could not convert size for %s got %s", path, response)
	}

	return &FileStat{
		UID:  uid,
		GID:  gid,
		Size: size,
		Name: parts[3],
	}, nil
}

// Rm removes a file from a remote
func (c *Client) Rm(targetPath string, pod *corev1.Pod,
	containerName string) (string, error) {

	targetPath = shellescape.Quote(targetPath)
	cmdArr := []string{"/bin/sh", "-c", "rm -rf " + targetPath}
	fmt.Println(strings.Join(cmdArr, " "))
	return c.ExecSync(pod, containerName, cmdArr, nil)
}

// tarMake is not used, but reserved for the future
func tarMake(srcPath, destPath string, writer io.Writer) error {
	// TODO: use compression here?
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)
	return tarMakeRecursive(path.Dir(srcPath), path.Base(srcPath), path.Dir(destPath), path.Base(destPath), tarWriter)
}

// tarMakeRecursive is not used, but reserved for the future
func tarMakeRecursive(srcBase, srcFile, destBase, destFile string, tw *tar.Writer) error {
	srcPath := path.Join(srcBase, srcFile)
	matchedPaths, err := filepath.Glob(srcPath)
	if err != nil {
		return err
	}
	for _, fpath := range matchedPaths {
		stat, err := os.Lstat(fpath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			files, err := os.ReadDir(fpath)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				//case empty directory
				hdr, _ := tar.FileInfoHeader(stat, fpath)
				hdr.Name = destFile
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
			}
			for _, f := range files {
				if err := tarMakeRecursive(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), tw); err != nil {
					return err
				}
			}
			return nil
		} else if stat.Mode()&os.ModeSymlink != 0 {
			//case soft link
			hdr, _ := tar.FileInfoHeader(stat, fpath)
			target, err := os.Readlink(fpath)
			if err != nil {
				return err
			}

			hdr.Linkname = target
			hdr.Name = destFile
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			//case regular file or other file type like pipe
			hdr, err := tar.FileInfoHeader(stat, fpath)
			if err != nil {
				return err
			}
			hdr.Name = destFile

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			f, err := os.Open(fpath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
			return f.Close()
		}
	}
	return nil
}

type PortForwardRequest struct {
	LocalPort    int // LocalPort is the local port that will be selected to expose the PodPort
	PodName      string
	PodNamespace string
	PodPort      int // PodPort is the target port for the pod
}

// It is to forward port, and return the forwarder.
func (c *Client) PortForward(req *PortForwardRequest) (*portforward.ForwardedPort, error) {
	// get the pod
	pod, err := c.PodGetByName(req.PodNamespace, req.PodName)
	if err != nil {
		return nil, err
	}

	// check the status
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("unable to forward port because pod %s is not running. Current status=%v", req.PodName, pod.Status.Phase)
	}

	// make the dialer to establish port forwarding
	kubeAPIUrl, err := url.Parse(c.Config.Host)
	if err != nil {
		return nil, err
	}
	kubeAPIUrl.Path = path.Join(
		"api", "v1",
		"namespaces", req.PodNamespace,
		"pods", req.PodName,
		"portforward",
	)
	transport, upgrader, err := spdy.RoundTripperFor(c.Config)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, kubeAPIUrl)

	// make the stop and ready channels
	stopCh := make(chan struct{}, 1)
	readyCh := make(chan struct{})

	// make pipes
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()

	// TODO when should I close these pipes?
	fw, err := portforward.New(dialer,
		[]string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)},
		stopCh,
		readyCh,
		outW,
		errW)
	if err != nil {
		return nil, err
	}
	// fw.GetPorts()

	// send all output to logs
	// TODO this isn't quite what we want
	go StreamAllToLog(fmt.Sprintf("%s/%s: ", req.PodNamespace, req.PodName), outR, errR)

	// stop forwarding if the os closes
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	go func() {
		<-signals
		if stopCh != nil {
			close(stopCh)
		}
	}()

	// forward
	go func() {
		if err := fw.ForwardPorts(); err != nil {
			core.Log.Fatalf("could not forward port: %v", err)
		}
	}()

	// wait on the ready channel
	<-readyCh

	// get the forwarded ports
	ports, err := fw.GetPorts()
	if err != nil {
		return nil, err
	}
	port := ports[0]
	return &port, nil
}
