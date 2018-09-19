package embeddedetcd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/coreos/etcd/embed"
	"github.com/juju/errors"
)

// Setup will embed config with the given parameters
func Setup(cfg *embed.Config, clusterState string, curls []url.URL, purls []url.URL) {
	cfg.ClusterState = clusterState
	cfg.LCUrls, cfg.ACUrls = curls, curls
	cfg.LPUrls, cfg.APUrls = purls, purls
	cfg.InitialCluster = ""
	for i := range purls {
		cfg.InitialCluster += ",default=" + purls[i].String()
	}
	cfg.InitialCluster = cfg.InitialCluster[1:]
}

// Start creates and instance of embedded ETCD
func Start(datafile string, bootstrap bool) (string, *embed.Etcd, error) {
	cfg := embed.NewConfig()

	urls := createUnixURLs(2)
	Setup(cfg, "new", []url.URL{urls[0]}, []url.URL{urls[1]})

	cfg.Dir = filepath.Join(os.TempDir(), datafile)
	if bootstrap {
		os.RemoveAll(cfg.Dir)
	}
	etcd, err := embed.StartEtcd(cfg)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	<-etcd.Server.ReadyNotify() // wait for e.Server to join the cluster

	return cfg.Dir, etcd, nil
}

func createUnixURLs(n int) (urls []url.URL) {
	for i := 0; i < n; i++ {
		u, _ := url.Parse(fmt.Sprintf("unix://localhost:%d%06d", os.Getpid(), i))
		urls = append(urls, *u)
	}
	return
}
