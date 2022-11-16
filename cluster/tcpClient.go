package cluster

import (
	"encoding/json"
	"log"
	"net"
	"os"
)

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

func SendTCPRequest(message []byte, servAddr string) Command {
	var command Command

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		println("Dial failed:", err.Error())
		os.Exit(1)
	}

	_, err = conn.Write(message)
	if err != nil {
		println("Write to server failed:", err.Error())
		os.Exit(1)
	}

	buffer := make([]byte, 1024)

	length, err := conn.Read(buffer)
	if err != nil {
		println("Write to server failed:", err.Error())
		os.Exit(1)
	}

	println("Reply from server=", string(buffer))

	conn.Close()
	buffer = buffer[:length]

	err = json.Unmarshal(buffer, &command)
	return command
}
