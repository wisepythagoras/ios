package consensus

import (
	"github.com/golang/glog"
	"github.com/heidi-ann/ios/msgs"
	"reflect"
)

// returns true if successful
// start index is inclusive and end index is exclusive
func runRecoveryCoordinator(view int, startIndex int, endIndex int, peerNet *msgs.PeerNet, config ConfigAll) bool {
	if startIndex == endIndex {
		return true
	} else if endIndex < startIndex {
		glog.Fatal("Invalid recovery range ", startIndex, endIndex)
	}
	glog.Info("Starting recovery for indexes ", startIndex, " to ", endIndex)

	// dispatch query to all
	query := msgs.QueryRequest{config.ID, view, startIndex, endIndex}
	peerNet.OutgoingBroadcast.Requests.Query <- query

	// collect responses
	noopEntry := msgs.Entry{0, false, []msgs.ClientRequest{noop}}
	candidates := make([]msgs.Entry, endIndex-startIndex)
	for i := 0; i < endIndex-startIndex; i++ {
		candidates[i] = noopEntry
	}

	//check only one response is received per sender, index= node ID

	for replied := make([]bool, config.N); !config.Quorum.checkRecoveryQuorum(replied); {
		msg := <-peerNet.Incoming.Responses.Query
		if msg.Request == query {

			// check this is not a duplicate
			if replied[msg.Response.SenderID] {
				glog.Warning("Response already received from ", msg.Response.SenderID)
			} else {
				// check view
				if msg.Response.View < view {
					glog.Fatal("Reply view is < current view, this should not have occurred")
				}

				if view < msg.Response.View {
					glog.Warning("Stepping down from recovery coordinator")
					return false
				}

				res := msg.Response
				replied[msg.Response.SenderID] = true

				for i := 0; i < endIndex-startIndex; i++ {
					if !reflect.DeepEqual(res.Entries[i], msgs.Entry{}) {
						// if committed, then done
						if res.Entries[i].Committed {
							candidates[i] = res.Entries[i]
							// TODO: add early exit here
						}

						// if first entry, then new candidate
						if reflect.DeepEqual(candidates[i], noopEntry) {
							candidates[i] = res.Entries[i]
						}

						// if higher view then candidate then new candidate
						if res.Entries[i].View > candidates[i].View {
							candidates[i] = res.Entries[i]
						}

						// if same view and different requests then panic!
						if res.Entries[i].View == candidates[i].View && !reflect.DeepEqual(res.Entries[i].Requests, candidates[i].Requests) {
							glog.Fatal("Same index has been issued more then once", res.Entries[i].Requests, candidates[i].Requests)
						}
					} else {
						glog.V(1).Info("Log entry at index ", i, " on node ID ", msg.Response.SenderID, " is missing")
					}
				}
			}
		}
	}
	glog.Info("New view phase is finished")

	// set the next view and marked as uncommitted
	// TODO: add shortcut to skip prepare phase is entries are already committed.
	for i := 0; i < endIndex-startIndex; i++ {
		candidates[i] = msgs.Entry{view, false, candidates[i].Requests}
	}

	coord := msgs.CoordinateRequest{config.ID, view, startIndex, endIndex, true, candidates}
	peerNet.OutgoingUnicast[config.ID].Requests.Coordinate <- coord
	<-peerNet.Incoming.Responses.Coordinate
	// TODO: check msg replies to the msg we just sent

	glog.Info("Recovery completed for indexes ", startIndex, " to ", endIndex)
	return true
}
