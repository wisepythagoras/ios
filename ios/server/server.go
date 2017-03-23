// Package server is the entry point to run an Ios server
package server

import (
	"github.com/golang/glog"
	"github.com/heidi-ann/ios/config"
	"github.com/heidi-ann/ios/consensus"
	"github.com/heidi-ann/ios/msgs"
	"github.com/heidi-ann/ios/net"
	"github.com/heidi-ann/ios/storage"
	"strings"
)

// RunIos id conf diskPath is the main entry point of Ios server
// It does not return
func RunIos(id int, conf config.ServerConfig, diskPath string) {
	// check ID
	if id >= len(conf.Peers.Address) {
		glog.Fatal("Node ID is ", id, " but is configured with a ", len(conf.Peers.Address), " node cluster")
	}

	// setup iO
	// TODO: remove this hardcoded limit on channel size
	iO := msgs.MakeIo(2000, len(conf.Peers.Address))

	// setup persistent storage
	found, view, log, index, state := storage.RestoreStorage(
		diskPath, conf.Options.Length, conf.Options.Application)
	var store msgs.Storage
	if conf.Unsafe.DumpPersistentStorage {
		store = msgs.MakeDummyStorage()
	} else {
		store = storage.MakeFileStorage(diskPath, conf.Unsafe.PersistenceMode)
	}

	// setup peers & clients
	failureDetector := msgs.NewFailureNotifier(len(conf.Peers.Address))
	net.SetupPeers(id, conf.Peers.Address, iO, failureDetector)
	net.SetupClients(strings.Split(conf.Clients.Address[id], ":")[1], state,iO)

	// configure consensus algorithms
	quorum := consensus.NewQuorum(conf.Options.QuorumSystem, len(conf.Peers.Address))
	configuration := consensus.Config{
		id,
		len(conf.Peers.Address),
		conf.Options.Length,
		conf.Options.BatchInterval,
		conf.Options.MaxBatch,
		conf.Options.DelegateReplication,
		conf.Options.WindowSize,
		conf.Options.SnapshotInterval,
		quorum,
		conf.Options.IndexExclusivity}

	// setup consensus algorithm
	if !found {
		glog.Info("Starting fresh consensus instance")
		consensus.Init(iO, configuration, state, failureDetector, store)
	} else {
		glog.Info("Restoring consensus instance")
		consensus.Recover(iO, configuration, view, log, state, index, failureDetector, store)
	}
}
