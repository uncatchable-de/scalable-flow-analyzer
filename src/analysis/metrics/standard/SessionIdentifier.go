package standard

// Analyzes Flows to identify sessions.

import (
	"scalable-flow-analyzer/flows"
	"scalable-flow-analyzer/metrics/common"
	"fmt"
	"sort"
	"sync"
	"time"
)

type sessionFlow struct {
	start        int64
	end          int64
	serverAddr   uint64
	clusterIndex int
}

type session struct {
	start               int64
	end                 int64
	sessionClusterIndex int
	flows               []*sessionFlow
}

type userSessionsStruct struct {
	sessions         []*session
	userClusterIndex int
	mutex            sync.Mutex
}

type protocolSessionsStruct struct {
	mutex         sync.Mutex
	protocol      common.Protocol
	usersSessions map[uint64]*userSessionsStruct // maps from user to sessions
}

type sessionIdentifier struct {
	mutex                    sync.Mutex
	clusterController        *ClusterController
	sessions                 map[common.ProtocolKeyType]*protocolSessionsStruct
	sessionTimeout           int64
	registeredSessionMetrics []SessionMetric
}

func newSessionIdentifier(sessionTimeout int64, clusterController *ClusterController) *sessionIdentifier {
	var si = &sessionIdentifier{
		sessions:          make(map[common.ProtocolKeyType]*protocolSessionsStruct),
		sessionTimeout:    sessionTimeout,
		clusterController: clusterController,
	}
	return si
}

func (si *sessionIdentifier) registerSessionMetric(metric SessionMetric) {
	si.registeredSessionMetrics = append(si.registeredSessionMetrics, metric)
}

// ForceFlush will flush all sessions to the metrics
// It will sort all flows within a session first.
// We do it here and not at insert (in the onFlush method)
// since this proved to be more time efficient.
func (si *sessionIdentifier) forceFlush() {
	var wg sync.WaitGroup
	wg.Add(len(si.sessions))
	for protocolKey := range si.sessions {
		go func(protocol common.Protocol, usersSessions map[uint64]*userSessionsStruct) {

			startTime := time.Now()
			fmt.Println("Starting Protocol:", protocol.GetProtocolString(), len(usersSessions), startTime)

			// Do sorting of flows and clustering in parallel. Needs with multithreading 15seconds on caida dataset with 128cores.
			// Without parallelization needs over 4 hours
			var wgUsers sync.WaitGroup
			wgUsers.Add(len(usersSessions))
			for userAddress, userSessions := range usersSessions {
				go func(userSessions *userSessionsStruct, userAddress uint64, wgUsers *sync.WaitGroup) {
					for _, session := range userSessions.sessions {
						// Sort flows
						sort.Slice(session.flows, func(i, j int) bool {
							return session.flows[i].start < session.flows[j].start
						})
						si.clusterController.CollectAndSetSessionClusterIndex(session, userAddress, &protocol)
					}

					si.clusterController.CollectAndSetUserClusterIndex(userSessions, userAddress, &protocol)
					wgUsers.Done()
				}(userSessions, userAddress, &wgUsers)
			}
			wgUsers.Wait()
			fmt.Println("Done sorting flows:", protocol.GetProtocolString(), time.Now(), time.Since(startTime))

			// Do metrics in parallel (can be further optimized, but only needs few seconds on caida dataset)
			fmt.Println(protocol.GetProtocolString(), len(si.registeredSessionMetrics))
			var wgMetrics sync.WaitGroup
			wgMetrics.Add(len(si.registeredSessionMetrics) * len(usersSessions))
			for _, metric := range si.registeredSessionMetrics {
				newMetric := metric
				for userAddress, user := range usersSessions {
					go func(metric SessionMetric, userAddress uint64, user *userSessionsStruct) {
						newMetric.OnFlush(protocol, userAddress, user)
						wgMetrics.Done()
					}(newMetric, userAddress, user)
				}
			}
			wgMetrics.Wait()
			fmt.Println("Done metrics:", protocol.GetProtocolString(), time.Now(), time.Since(startTime))
			wg.Done()
		}(si.sessions[protocolKey].protocol, si.sessions[protocolKey].usersSessions)
	}
	wg.Wait()
	si.sessions = make(map[common.ProtocolKeyType]*protocolSessionsStruct)
}

// The sessionIdentifier sorts all flushed connections based on protocol and client. The flows can (and will) arrive out of order.
// Therefore the sessionIdentifier stores them in a sorted list and flushes them at the end to the corresponding metrics.
func (si *sessionIdentifier) onFlush(flow *flows.Flow) {
	var userKey = flow.ClientAddr
	var protocol = common.GetProtocol(flow)
	var flowStart = flow.Packets[0].Timestamp
	var flowEnd = flow.Packets[len(flow.Packets)-1].Timestamp
	var newSessionFlow = &sessionFlow{start: flowStart, end: flowEnd, serverAddr: flow.ServerAddr, clusterIndex: flow.ClusterIndex}
	var protSessions *protocolSessionsStruct
	var ok bool

	si.mutex.Lock()
	if protSessions, ok = si.sessions[protocol.ProtocolKey]; !ok {
		protSessions = &protocolSessionsStruct{protocol: protocol, usersSessions: make(map[uint64]*userSessionsStruct)}
		si.sessions[protocol.ProtocolKey] = protSessions
	}
	si.mutex.Unlock()
	protSessions.mutex.Lock()
	if userSessions, ok := protSessions.usersSessions[userKey]; ok {
		protSessions.mutex.Unlock()
		userSessions.mutex.Lock()
		defer userSessions.mutex.Unlock()
		// Search first session which starts after this flow
		sessionIdx := sort.Search(len(userSessions.sessions), func(i int) bool { return userSessions.sessions[i].start > flowStart })
		if sessionIdx > 0 {
			// Previous session: the last one which starts before the current flow
			previousSession := userSessions.sessions[sessionIdx-1]

			switch {
			// If previous session covers current flow: add flow
			case previousSession.end >= flowEnd:
				previousSession.flows = append(previousSession.flows, newSessionFlow)
				return

				// If previous session can be extended: extend it and add flow
			case flowStart-previousSession.end <= si.sessionTimeout:
				previousSession.flows = append(previousSession.flows, newSessionFlow)
				previousSession.end = flowEnd
				sessionIdx--

				// If previous session is already timed out: add new session at sessionIdx in ordered list
			default:
				// The check if next session can be merged happens below
				// Insert at i https://github.com/golang/go/wiki/SliceTricks
				userSessions.sessions = append(userSessions.sessions, nil)
				copy(userSessions.sessions[sessionIdx+1:], userSessions.sessions[sessionIdx:])
				userSessions.sessions[sessionIdx] = &session{start: flowStart, end: flowEnd, flows: []*sessionFlow{newSessionFlow}}
			}

			// add new session at beginning
		} else {
			userSessions.sessions = append(userSessions.sessions, nil)
			copy(userSessions.sessions[1:], userSessions.sessions[0:])
			userSessions.sessions[0] = &session{start: flowStart, end: flowEnd, flows: []*sessionFlow{newSessionFlow}}
		}

		// Check if new session (at sessionIdx) can be merged with the next sessions
		for true {
			// Check if there is at least one session afterwards
			if len(userSessions.sessions) <= sessionIdx+1 {
				break
			}

			// Break if the new session cannot be merged with the next session
			if userSessions.sessions[sessionIdx+1].start-userSessions.sessions[sessionIdx].end > si.sessionTimeout {
				break
			}

			// new end is maximum of the two merging sessions
			if userSessions.sessions[sessionIdx+1].end > userSessions.sessions[sessionIdx].end {
				userSessions.sessions[sessionIdx].end = userSessions.sessions[sessionIdx+1].end
			}
			// append flows
			userSessions.sessions[sessionIdx].flows = append(userSessions.sessions[sessionIdx].flows, userSessions.sessions[sessionIdx+1].flows...)
			// Delete session i+1
			// https://github.com/golang/go/wiki/SliceTricks
			copy(userSessions.sessions[sessionIdx+1:], userSessions.sessions[sessionIdx+2:])
			userSessions.sessions[len(userSessions.sessions)-1] = nil
			userSessions.sessions = userSessions.sessions[:len(userSessions.sessions)-1]
		}
	} else {
		// add new session
		protSessions.usersSessions[userKey] = &userSessionsStruct{
			sessions: []*session{{start: flowStart, end: flowEnd, flows: []*sessionFlow{newSessionFlow}}},
		}
		protSessions.mutex.Unlock()
	}
}

func (si *sessionIdentifier) OnTCPFlush(flow *flows.TCPFlow) {
	si.onFlush(&flow.Flow)
}

func (si *sessionIdentifier) OnUDPFlush(flow *flows.UDPFlow) {
	si.onFlush(&flow.Flow)
}

func (si *sessionIdentifier) PrintStatistic(verbose bool) {

}
