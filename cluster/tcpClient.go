package cluster

import (
	"encoding/json"
	"log"
	"net"
)

// I used this function when all nodes replicated data
func SendCommandToPeers(command Command) {
	node := GetNode()
	byteMsg, err := json.Marshal(command)
	if err != nil {
		log.Fatal(err)
	}
	//log.Println(node.Peers)
	//log.Println(node.Name)
	for _, peer := range node.Peers {
		SendTCPRequest(byteMsg, peer.Name+peer.TcpPort)
	}
}

// Used to sent TCP requests
func SendTCPRequest(message []byte, servAddr string) (Command, error) {
	var command Command

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		//println("ResolveTCPAddr failed:", err.Error())
		return command, err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		//println("Dial failed:", err.Error())
		return command, err

	}
	defer conn.Close()

	_, err = conn.Write(message)
	if err != nil {
		//println("Write to tcp client failed:", err.Error())
		return command, err
	}

	buffer := make([]byte, 1024)

	length, err := conn.Read(buffer)
	if err != nil {
		//println("Read from server failed:", err.Error())
		return command, err
	}

	//println("Reply from server=", string(buffer))

	err = conn.Close()
	if err != nil {
		//println("Closing tcp connection failed:", err.Error())
		return command, err
	}

	buffer = buffer[:length]

	err = json.Unmarshal(buffer, &command)
	if err != nil {
		//println("Unmarshalling failed:", err.Error())
		return command, err
	}
	return command, nil
}
