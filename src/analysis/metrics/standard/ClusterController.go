package standard

import (
	"analysis/flows"
	"analysis/metrics/common"
	"analysis/utils"
	"clustering/clustering"
	"clustering/dataformat"
	"log"
	"path"
	"sync"

	"github.com/uncatchable-de/goml/cluster"
)

const userDirectory = "user"
const sessionDirectory = "session"
const flowDirectory = "flow"
const rrpDirectory = "rrp"

// ClusterController has two tasks:
// - Collecting flow information (metrics) which are stored to a file and can be later used for classification learning
// - Identify to which cluster a flow belongs, based on the external trained model
type ClusterController struct {
	rrpModel           map[common.ProtocolKeyType]Model
	flowModel          map[common.ProtocolKeyType]Model
	sessionModel       map[common.ProtocolKeyType]Model
	userModel          map[common.ProtocolKeyType]Model
	metric             *Metric
	rrpsInfo           map[common.ProtocolKeyType]*RRPsInfos
	rrpsInfoMutex      sync.Mutex
	flowsInfo          map[common.ProtocolKeyType]*FlowsInfos
	flowsInfoMutex     sync.Mutex
	sessionsInfo       map[common.ProtocolKeyType]*SessionsInfos
	sessionsInfoMutex  sync.Mutex
	usersInfo          map[common.ProtocolKeyType]*UsersInfos
	usersInfoMutex     sync.Mutex
	useClusters        bool
	collectClusterInfo bool
	infoFilesPath      string
}

type Model struct {
	model   *cluster.KMeans
	options *clustering.SaveOptions
}

type RRPsInfos struct {
	RRPs     *dataformat.RRPs
	Protocol common.Protocol
}

type FlowsInfos struct {
	Flows    *dataformat.Flows
	Protocol common.Protocol
}

type SessionsInfos struct {
	Sessions *dataformat.Sessions
	Protocol common.Protocol
}

type UsersInfos struct {
	Users    *dataformat.Users
	Protocol common.Protocol
}

// DefaultClusterIndex is used whenever a metric does not support clustering, or no clustering is used
const DefaultClusterIndex = 0

func NewClusterController(metric *Metric, infoPath, modelPath string) *ClusterController {
	cc := &ClusterController{
		metric:       metric,
		rrpModel:     make(map[common.ProtocolKeyType]Model),
		rrpsInfo:     make(map[common.ProtocolKeyType]*RRPsInfos),
		flowModel:    make(map[common.ProtocolKeyType]Model),
		flowsInfo:    make(map[common.ProtocolKeyType]*FlowsInfos),
		sessionModel: make(map[common.ProtocolKeyType]Model),
		sessionsInfo: make(map[common.ProtocolKeyType]*SessionsInfos),
		userModel:    make(map[common.ProtocolKeyType]Model),
		usersInfo:    make(map[common.ProtocolKeyType]*UsersInfos),
	}
	switch infoPath {
	case "":
		cc.collectClusterInfo = false
	default:
		cc.collectClusterInfo = true
		cc.infoFilesPath = infoPath
		if !utils.DirectoryExists(path.Join(infoPath, rrpDirectory)) {
			utils.CreateDir(path.Join(infoPath, rrpDirectory))
		}
		if !utils.DirectoryExists(path.Join(infoPath, flowDirectory)) {
			utils.CreateDir(path.Join(infoPath, flowDirectory))
		}
		if !utils.DirectoryExists(path.Join(infoPath, sessionDirectory)) {
			utils.CreateDir(path.Join(infoPath, sessionDirectory))
		}
		if !utils.DirectoryExists(path.Join(infoPath, userDirectory)) {
			utils.CreateDir(path.Join(infoPath, userDirectory))
		}
	}

	switch modelPath {
	case "":
		cc.useClusters = false
	default:
		cc.useClusters = true

		// Load User models
		userModelDirectory := path.Join(modelPath, userDirectory)
		if !utils.DirectoryExists(userModelDirectory) {
			log.Fatalln("Directory for user cluster model does not exist:", userModelDirectory)
		}
		for _, filepath := range utils.GetFilesInPath(userModelDirectory, "json") {
			model, options, err := clustering.LoadModel(filepath)
			if err != nil {
				log.Fatalln("Error while loading user Cluster Model from", filepath, err)
			}
			cc.userModel[common.GetProtocolKey(utils.GetFilename(filepath, false))] = Model{
				model:   model,
				options: options,
			}
		}

		// Load Session models
		sessionModelDirectory := path.Join(modelPath, sessionDirectory)
		if !utils.DirectoryExists(sessionModelDirectory) {
			log.Fatalln("Directory for session cluster model does not exist:", sessionModelDirectory)
		}
		for _, filepath := range utils.GetFilesInPath(sessionModelDirectory, "json") {
			model, options, err := clustering.LoadModel(filepath)
			if err != nil {
				log.Fatalln("Error while loading session Cluster Model from", filepath, err)
			}
			cc.sessionModel[common.GetProtocolKey(utils.GetFilename(filepath, false))] = Model{
				model:   model,
				options: options,
			}
		}

		// Load Flow models
		flowModelDirectory := path.Join(modelPath, flowDirectory)
		if !utils.DirectoryExists(flowModelDirectory) {
			log.Fatalln("Directory for flow cluster models does not exist:", flowModelDirectory)
		}
		for _, filepath := range utils.GetFilesInPath(flowModelDirectory, "json") {
			model, options, err := clustering.LoadModel(filepath)
			if err != nil {
				log.Fatalln("Error while loading flow Cluster Model from", filepath, err)
			}
			cc.flowModel[common.GetProtocolKey(utils.GetFilename(filepath, false))] = Model{
				model:   model,
				options: options,
			}
		}

		// Load RRP models
		rrpModelDirectory := path.Join(modelPath, rrpDirectory)
		if !utils.DirectoryExists(rrpModelDirectory) {
			log.Fatalln("Directory for rrp cluster models does not exist:", rrpModelDirectory)
		}
		for _, filepath := range utils.GetFilesInPath(rrpModelDirectory, "json") {
			model, options, err := clustering.LoadModel(filepath)
			if err != nil {
				log.Fatalln("Error while loading flow Cluster Model from", filepath, err)
			}
			cc.rrpModel[common.GetProtocolKey(utils.GetFilename(filepath, false))] = Model{
				model:   model,
				options: options,
			}
		}
	}
	return cc
}

func (cc *ClusterController) GetNumberOfRRPClusters(protocolKey common.ProtocolKeyType) int {
	if !cc.useClusters {
		return 1
	}
	return len(cc.rrpModel[protocolKey].model.Centroids)
}

func (cc *ClusterController) GetNumberOfFlowClusters(protocolKey common.ProtocolKeyType) int {
	if !cc.useClusters {
		return 1
	}
	return len(cc.flowModel[protocolKey].model.Centroids)
}

func (cc *ClusterController) GetNumberOfSessionClusters(protocolKey common.ProtocolKeyType) int {
	if !cc.useClusters {
		return 1
	}
	return len(cc.sessionModel[protocolKey].model.Centroids)
}

func (cc *ClusterController) GetNumberOfUserClusters(protocolKey common.ProtocolKeyType) int {
	if !cc.useClusters {
		return 1
	}
	return len(cc.userModel[protocolKey].model.Centroids)
}

// Returns the index of the cluster to which the rrps belongs
func (cc *ClusterController) CollectAndSetRRPClusterIndex(flow *flows.Flow, reqRes []*common.RequestResponse) {
	if !cc.useClusters && !cc.collectClusterInfo {
		for i := 0; i < len(reqRes); i++ {
			reqRes[i].ClusterIndex = DefaultClusterIndex
		}
		return
	}

	protocolKey := common.GetProtocol(flow).ProtocolKey
	rrpInfos := cc.getRRPInfos(flow, reqRes)

	// Predict
	if !cc.useClusters {
		for i := 0; i < len(reqRes); i++ {
			reqRes[i].ClusterIndex = DefaultClusterIndex
		}
	} else {
		if _, ok := cc.rrpModel[protocolKey]; !ok {
			log.Fatalln("No RRP Model for prediction")
		}
		for i := 0; i < len(reqRes); i++ {
			rrpData := clustering.GetDataOfRRP(rrpInfos[i])
			clustering.ScaleData(rrpData, cc.rrpModel[protocolKey].options)
			clusterIdx, err := cc.rrpModel[protocolKey].model.Predict(rrpData)
			if err != nil {
				log.Fatalln("Error while predicting values", err)
			}
			// Predict always returns a vector. First element is the cluster index
			reqRes[i].ClusterIndex = int(clusterIdx[0])
		}
	}

	if !cc.collectClusterInfo {
		return
	}

	cc.rrpsInfoMutex.Lock()
	if _, ok := cc.rrpsInfo[protocolKey]; !ok {
		cc.rrpsInfo[protocolKey] = &RRPsInfos{
			Protocol: common.GetProtocol(flow),
			RRPs:     &dataformat.RRPs{Rrps: make([]*dataformat.RRP, 0)},
		}
	}
	cc.rrpsInfo[protocolKey].RRPs.Rrps = append(cc.rrpsInfo[protocolKey].RRPs.Rrps, rrpInfos...)
	cc.rrpsInfoMutex.Unlock()
}

// Returns the index of the cluster to which the flow belongs
func (cc *ClusterController) CollectAndSetFlowClusterIndex(flow *flows.Flow, reqRes []*common.RequestResponse) {
	if !cc.useClusters && !cc.collectClusterInfo {
		flow.ClusterIndex = DefaultClusterIndex
	}

	protocolKey := common.GetProtocol(flow).ProtocolKey
	flowInfos := cc.getFlowInfo(flow, reqRes)

	// Predict
	if !cc.useClusters {
		flow.ClusterIndex = DefaultClusterIndex
	} else {
		if _, ok := cc.flowModel[protocolKey]; !ok {
			log.Fatalln("No Flow Model for prediction")
		}
		flowData := clustering.GetDataOfFlow(flowInfos)
		clustering.ScaleData(flowData, cc.flowModel[protocolKey].options)
		clusterIdx, err := cc.flowModel[protocolKey].model.Predict(flowData)
		if err != nil {
			log.Fatalln("Error while predicting values", err)
		}
		// Predict always returns a vector. First element is the cluster index
		flow.ClusterIndex = int(clusterIdx[0])
	}

	cc.flowsInfoMutex.Lock()
	if _, ok := cc.flowsInfo[protocolKey]; !ok {
		cc.flowsInfo[protocolKey] = &FlowsInfos{
			Protocol: common.GetProtocol(flow),
			Flows:    &dataformat.Flows{Flows: make([]*dataformat.Flow, 0)},
		}
	}
	cc.flowsInfo[protocolKey].Flows.Flows = append(cc.flowsInfo[protocolKey].Flows.Flows, flowInfos)
	cc.flowsInfoMutex.Unlock()
}

// Returns the index of the cluster to which the session belongs
func (cc *ClusterController) CollectAndSetSessionClusterIndex(session *session, clientAddress uint64, protocol *common.Protocol) {
	if !cc.useClusters && !cc.collectClusterInfo {
		session.sessionClusterIndex = DefaultClusterIndex
		return
	}
	protocolKey := protocol.ProtocolKey
	sessionInfos := cc.getSessionInfo(session, clientAddress)

	// Predict
	if !cc.useClusters {
		session.sessionClusterIndex = DefaultClusterIndex
	} else {
		if _, ok := cc.sessionModel[protocolKey]; !ok {
			log.Fatalln("No Session Model for prediction")
		}
		sessionData := clustering.GetDataOfSession(sessionInfos)
		clustering.ScaleData(sessionData, cc.sessionModel[protocolKey].options)
		clusterIdx, err := cc.sessionModel[protocolKey].model.Predict(sessionData)
		if err != nil {
			log.Fatalln("Error while predicting values", err)
		}
		// Predict always returns a vector. First element is the cluster index
		session.sessionClusterIndex = int(clusterIdx[0])
	}

	if !cc.collectClusterInfo {
		return
	}
	// Collect Session Info
	cc.sessionsInfoMutex.Lock()
	if _, ok := cc.sessionsInfo[protocolKey]; !ok {
		cc.sessionsInfo[protocolKey] = &SessionsInfos{
			Protocol: *protocol,
			Sessions: &dataformat.Sessions{Sessions: make([]*dataformat.Session, 0)},
		}
	}
	cc.sessionsInfo[protocolKey].Sessions.Sessions = append(cc.sessionsInfo[protocolKey].Sessions.Sessions, sessionInfos)
	cc.sessionsInfoMutex.Unlock()
}

// Returns the index of the cluster to which the user belongs
func (cc *ClusterController) CollectAndSetUserClusterIndex(sessions *userSessionsStruct, clientAddress uint64, protocol *common.Protocol) {
	if !cc.useClusters && !cc.collectClusterInfo {
		sessions.userClusterIndex = DefaultClusterIndex
		return
	}
	protocolKey := protocol.ProtocolKey
	userInfos := cc.getUserInfo(sessions, clientAddress)

	// Predict
	if !cc.useClusters {
		sessions.userClusterIndex = DefaultClusterIndex
	} else {
		protocolKey := protocol.ProtocolKey
		if _, ok := cc.userModel[protocolKey]; !ok {
			log.Fatalln("No user Model for prediction")
		}
		userData := clustering.GetDataOfUser(userInfos)
		clustering.ScaleData(userData, cc.userModel[protocolKey].options)
		clusterIdx, err := cc.userModel[protocolKey].model.Predict(userData)
		if err != nil {
			log.Fatalln("Error while predicting values", err)
		}
		// Predict always returns a vector. First element is the cluster index
		sessions.userClusterIndex = int(clusterIdx[0])
	}

	if !cc.collectClusterInfo {
		return
	}

	cc.usersInfoMutex.Lock()
	if _, ok := cc.usersInfo[protocolKey]; !ok {
		cc.usersInfo[protocolKey] = &UsersInfos{
			Protocol: *protocol,
			Users:    &dataformat.Users{Users: make([]*dataformat.User, 0)},
		}
	}
	cc.usersInfo[protocolKey].Users.Users = append(cc.usersInfo[protocolKey].Users.Users, userInfos)
	cc.usersInfoMutex.Unlock()
}

func (cc *ClusterController) PersistRRPInfos(clearMemory bool) {
	if !cc.collectClusterInfo {
		return
	}

	// For each protocol save flows
	for _, rrps := range cc.rrpsInfo {
		fname := path.Join(cc.infoFilesPath, rrpDirectory, rrps.Protocol.GetProtocolString()+".data")
		clustering.SaveRRPData(fname, rrps.RRPs, 5000)
	}

	if clearMemory {
		cc.rrpsInfoMutex.Lock()
		cc.rrpsInfo = make(map[common.ProtocolKeyType]*RRPsInfos)
		cc.rrpsInfoMutex.Unlock()
	}
}

func (cc *ClusterController) PersistFlowInfos(clearMemory bool) {
	if !cc.collectClusterInfo {
		return
	}

	// For each protocol save flows
	for _, flowInfo := range cc.flowsInfo {
		fileName := path.Join(cc.infoFilesPath, flowDirectory, flowInfo.Protocol.GetProtocolString()+".data")
		clustering.SaveFlowData(fileName, flowInfo.Flows, 5000)
	}

	if clearMemory {
		cc.flowsInfoMutex.Lock()
		cc.flowsInfo = make(map[common.ProtocolKeyType]*FlowsInfos)
		cc.flowsInfoMutex.Unlock()
	}
}

func (cc *ClusterController) PersistSessionInfos(clearMemory bool) {
	if !cc.collectClusterInfo {
		return
	}

	// For each protocol save flows
	for _, sessions := range cc.sessionsInfo {
		fname := path.Join(cc.infoFilesPath, sessionDirectory, sessions.Protocol.GetProtocolString()+".data")
		clustering.SaveSessionData(fname, sessions.Sessions, 5000)
	}

	if clearMemory {
		cc.sessionsInfoMutex.Lock()
		cc.sessionsInfo = make(map[common.ProtocolKeyType]*SessionsInfos)
		cc.sessionsInfoMutex.Unlock()
	}
}

func (cc *ClusterController) PersistUserInfos(clearMemory bool) {
	if !cc.collectClusterInfo {
		return
	}

	// For each protocol save flows
	for _, users := range cc.usersInfo {
		fname := path.Join(cc.infoFilesPath, userDirectory, users.Protocol.GetProtocolString()+".data")
		clustering.SaveUserData(fname, users.Users, 5000)
	}

	if clearMemory {
		cc.usersInfoMutex.Lock()
		cc.usersInfo = make(map[common.ProtocolKeyType]*UsersInfos)
		cc.usersInfoMutex.Unlock()
	}
}

func getUnivariateOfBivariate(values [][]int) []int {
	var uniVariateValues []int
	for i := range values {
		uniVariateValues = append(uniVariateValues, values[i][1])
	}
	return uniVariateValues
}

// Returns the RRP Infos for a flow
func (cc *ClusterController) getRRPInfos(flow *flows.Flow, reqRes []*common.RequestResponse) []*dataformat.RRP {
	reqSizes, resSizes := cc.metric.MetricSize.calc(reqRes)
	var outputRRPs []*dataformat.RRP
	for i := 0; i < len(reqRes); i++ {
		outputRRPs = append(outputRRPs, &dataformat.RRP{
			RequestSize:  int64(reqSizes[i]),
			ResponseSize: int64(resSizes[i]),
		})
	}
	return outputRRPs
}

// Returns the flow Info for a flow
func (cc *ClusterController) getFlowInfo(flow *flows.Flow, reqRes []*common.RequestResponse) *dataformat.Flow {
	interReq := cc.metric.MetricInterRequest.calc(reqRes)
	interReqMean, interReqMin, interReqMax, interReqStdDev := utils.GetDistributionStats(getUnivariateOfBivariate(interReq))
	flowInfo := &dataformat.Flow{
		ServerAddress: flow.ServerAddr,
		NumRrp:        int64(len(reqRes)),
		InterReq: &dataformat.Distribution{Mean: interReqMean,
			Min:    int64(interReqMin),
			Max:    int64(interReqMax),
			StdDev: interReqStdDev,
		},
	}
	return flowInfo
}

// Returns the session Info for a session
func (cc *ClusterController) getSessionInfo(session *session, clientAddress uint64) *dataformat.Session {
	numServers := cc.metric.MetricNumServers.calc(session)
	numFlows := cc.metric.MetricNumFlows.calc(session)
	interFlowTimes := cc.metric.MetricInterFlowTimes.calc(session)
	interFlowTimesMean, interFlowTimesMin, interFlowTimesMax, interFlowTimesStdDev := utils.GetDistributionStats(interFlowTimes)
	sessionInfo := &dataformat.Session{
		ClientAddress: clientAddress,
		NumServers:    int64(numServers),
		NumFlows:      int64(numFlows),
		InterFlow: &dataformat.Distribution{Mean: interFlowTimesMean,
			Min:    int64(interFlowTimesMin),
			Max:    int64(interFlowTimesMax),
			StdDev: interFlowTimesStdDev,
		},
	}
	return sessionInfo
}

// Returns the user Info for a session
func (cc *ClusterController) getUserInfo(sessions *userSessionsStruct, clientAddress uint64) *dataformat.User {
	interSession := cc.metric.MetricInterSessions.calc(sessions)
	interSessionMean, interSessionMin, interSessionMax, interSessionStdDev := utils.GetDistributionStats(interSession)
	userInfo := &dataformat.User{
		ClientAddress: clientAddress,
		NumSessions:   int64(len(sessions.sessions)),
		InterSession: &dataformat.Distribution{Mean: interSessionMean,
			Min:    int64(interSessionMin),
			Max:    int64(interSessionMax),
			StdDev: interSessionStdDev,
		},
	}
	return userInfo
}
