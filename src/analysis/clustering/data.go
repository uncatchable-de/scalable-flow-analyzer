package clustering

import (
	"scalable-flow-analyzer/dataformat"
	"encoding/binary"
	"io"
	"log"
	"os"

	"github.com/golang/protobuf/proto"
)

func LoadRRPData(filename string) *dataformat.RRPs {
	allRRPs := &dataformat.RRPs{}
	// Open file
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	// Get Number of RRPs
	sizeBytes := make([]byte, 8)
	_, err = file.Read(sizeBytes)
	if err != nil {
		panic(err)
	}
	numRRPs := binary.BigEndian.Uint64(sizeBytes)
	allRRPs.Rrps = make([]*dataformat.RRP, numRRPs)

	var i int
	for true {
		// Read size of message
		n, err := file.Read(sizeBytes)
		if n == 0 && err == io.EOF {
			break
		}
		if n != 8 {
			log.Fatalln("Number of read bytes must be 8")
		}
		size := binary.BigEndian.Uint64(sizeBytes)

		// Read message of the specified size, unmarshal it and add it to result
		buf := make([]byte, size)
		_, err = file.Read(buf)
		if err != nil {
			log.Fatalln("Error at reading cluster data File", err)
		}
		rrps := &dataformat.RRPs{}
		err = proto.Unmarshal(buf, rrps)
		if err != nil {
			log.Fatalln("Errowr while unmarshaling cluster data", err)
		}
		for j := 0; j < len(rrps.Rrps); j++ {
			allRRPs.Rrps[i+j] = rrps.Rrps[j]
		}
		i += len(rrps.Rrps)
	}
	return allRRPs
}

func LoadFlowData(filename string) *dataformat.Flows {
	allFlows := &dataformat.Flows{}
	// Open file
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	// Get Number of Flows
	sizeBytes := make([]byte, 8)
	_, err = file.Read(sizeBytes)
	if err != nil {
		panic(err)
	}
	numFlows := binary.BigEndian.Uint64(sizeBytes)
	allFlows.Flows = make([]*dataformat.Flow, numFlows)

	var i int
	for true {
		// Read size of message
		n, err := file.Read(sizeBytes)
		if n == 0 && err == io.EOF {
			break
		}
		if n != 8 {
			log.Fatalln("Number of read bytes must be 8")
		}
		size := binary.BigEndian.Uint64(sizeBytes)

		// Read message of the specified size, unmarshal it and add it to result
		buf := make([]byte, size)
		_, err = file.Read(buf)
		if err != nil {
			log.Fatalln("Error at reading cluster data File", err)
		}
		flows := &dataformat.Flows{}
		err = proto.Unmarshal(buf, flows)
		if err != nil {
			log.Fatalln("Errowr while unmarshaling cluster data", err)
		}
		for j := 0; j < len(flows.Flows); j++ {
			allFlows.Flows[i+j] = flows.Flows[j]
		}
		i += len(flows.Flows)
	}
	return allFlows
}

func LoadSessionData(filename string) *dataformat.Sessions {
	allSessions := &dataformat.Sessions{}
	// Open file
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	// Get Number of Sessions
	sizeBytes := make([]byte, 8)
	_, err = file.Read(sizeBytes)
	if err != nil {
		panic(err)
	}
	numSessions := binary.BigEndian.Uint64(sizeBytes)
	allSessions.Sessions = make([]*dataformat.Session, numSessions)

	var i int
	for true {
		// Read size of message
		n, err := file.Read(sizeBytes)
		if n == 0 && err == io.EOF {
			break
		}
		if n != 8 {
			log.Fatalln("Number of read bytes must be 8")
		}
		size := binary.BigEndian.Uint64(sizeBytes)

		// Read message of the specified size, unmarshal it and add it to result
		buf := make([]byte, size)
		_, err = file.Read(buf)
		if err != nil {
			log.Fatalln("Error at reading cluster data File", err)
		}
		sessions := &dataformat.Sessions{}
		err = proto.Unmarshal(buf, sessions)
		if err != nil {
			log.Fatalln("Errowr while unmarshaling cluster data", err)
		}
		for j := 0; j < len(sessions.Sessions); j++ {
			allSessions.Sessions[i+j] = sessions.Sessions[j]
		}
		i += len(sessions.Sessions)
	}
	return allSessions
}

func LoadUserData(filename string) *dataformat.Users {
	allUsers := &dataformat.Users{}
	// Open file
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	// Get Number of Users
	sizeBytes := make([]byte, 8)
	_, err = file.Read(sizeBytes)
	if err != nil {
		panic(err)
	}
	numUsers := binary.BigEndian.Uint64(sizeBytes)
	allUsers.Users = make([]*dataformat.User, numUsers)

	var i int
	for true {
		// Read size of message
		n, err := file.Read(sizeBytes)
		if n == 0 && err == io.EOF {
			break
		}
		if n != 8 {
			log.Fatalln("Number of read bytes must be 8")
		}
		size := binary.BigEndian.Uint64(sizeBytes)

		// Read message of the specified size, unmarshal it and add it to result
		buf := make([]byte, size)
		_, err = file.Read(buf)
		if err != nil {
			log.Fatalln("Error at reading cluster data File", err)
		}
		users := &dataformat.Users{}
		err = proto.Unmarshal(buf, users)
		if err != nil {
			log.Fatalln("Errowr while unmarshaling cluster data", err)
		}
		for j := 0; j < len(users.Users); j++ {
			allUsers.Users[i+j] = users.Users[j]
		}
		i += len(users.Users)
	}
	return allUsers
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func SaveRRPData(filename string, rrps *dataformat.RRPs, chunksize int) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	var buf []byte
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, uint64(len(rrps.Rrps)))
	_, err = file.Write(sizeBytes)
	if err != nil {
		log.Fatalln("Error while writing number of rrps", err)
	}

	for i := 0; i < len(rrps.Rrps); i += chunksize {
		batch := rrps.Rrps[i:min(i+chunksize, len(rrps.Rrps))]
		batchRRPs := &dataformat.RRPs{Rrps: batch}
		buf, err = proto.Marshal(batchRRPs)
		if err != nil {
			log.Fatalln("Error while marshaling Batch", err)
		}
		// Write Size
		binary.BigEndian.PutUint64(sizeBytes, uint64(len(buf)))
		_, err = file.Write(sizeBytes)
		if err != nil {
			log.Fatalln("Error while writing size", err)
		}
		// Write message
		_, err = file.Write(buf)
		if err != nil {
			log.Fatalln("Error while writing marshaled message", err)
		}
	}
}

func SaveFlowData(filename string, flows *dataformat.Flows, chunksize int) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	var buf []byte
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, uint64(len(flows.Flows)))
	_, err = file.Write(sizeBytes)
	if err != nil {
		log.Fatalln("Error while writing number of flows", err)
	}

	for i := 0; i < len(flows.Flows); i += chunksize {
		batch := flows.Flows[i:min(i+chunksize, len(flows.Flows))]
		batchFlows := &dataformat.Flows{Flows: batch}
		buf, err = proto.Marshal(batchFlows)
		if err != nil {
			log.Fatalln("Error while marshaling Batch", err)
		}
		// Write Size
		binary.BigEndian.PutUint64(sizeBytes, uint64(len(buf)))
		_, err = file.Write(sizeBytes)
		if err != nil {
			log.Fatalln("Error while writing size", err)
		}
		// Write message
		_, err = file.Write(buf)
		if err != nil {
			log.Fatalln("Error while writing marshaled message", err)
		}
	}
}

func SaveSessionData(filename string, sessions *dataformat.Sessions, chunksize int) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	var buf []byte
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, uint64(len(sessions.Sessions)))
	_, err = file.Write(sizeBytes)
	if err != nil {
		log.Fatalln("Error while writing number of sessions", err)
	}

	for i := 0; i < len(sessions.Sessions); i += chunksize {
		batch := sessions.Sessions[i:min(i+chunksize, len(sessions.Sessions))]
		batchSessions := &dataformat.Sessions{Sessions: batch}
		buf, err = proto.Marshal(batchSessions)
		if err != nil {
			log.Fatalln("Error while marshaling Batch", err)
		}
		// Write Size
		binary.BigEndian.PutUint64(sizeBytes, uint64(len(buf)))
		_, err = file.Write(sizeBytes)
		if err != nil {
			log.Fatalln("Error while writing size", err)
		}
		// Write message
		_, err = file.Write(buf)
		if err != nil {
			log.Fatalln("Error while writing marshaled message", err)
		}
	}
}

func SaveUserData(filename string, users *dataformat.Users, chunksize int) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	var buf []byte
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, uint64(len(users.Users)))
	_, err = file.Write(sizeBytes)
	if err != nil {
		log.Fatalln("Error while writing number of users", err)
	}

	for i := 0; i < len(users.Users); i += chunksize {
		batch := users.Users[i:min(i+chunksize, len(users.Users))]
		batchUsers := &dataformat.Users{Users: batch}
		buf, err = proto.Marshal(batchUsers)
		if err != nil {
			log.Fatalln("Error while marshaling Batch", err)
		}
		// Write Size
		binary.BigEndian.PutUint64(sizeBytes, uint64(len(buf)))
		_, err = file.Write(sizeBytes)
		if err != nil {
			log.Fatalln("Error while writing size", err)
		}
		// Write message
		_, err = file.Write(buf)
		if err != nil {
			log.Fatalln("Error while writing marshaled message", err)
		}
	}
}
